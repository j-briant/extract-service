package extraction

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/airbusgeo/godal"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RequestParameters struct {
	Product    string `json:"product" binding:"required"`
	Perimeter  string `json:"perimeter" binding:"required"`
	Parameters string `json:"parameters" binding:"required"`
}

type WFSRequestParameters struct {
	BaseURL   string
	OGCServer string
	Service   string
	Version   string
	Request   string
	TypeNames []string
}

type SpatialFilter struct {
	SRID         string `json:"srid"`
	GeometryType string `json:"geometrytype"`
	Coordinates  string `json:"coordinates"`
}

type UUIDCSV struct {
	UUID   string   `json:"uuid"`
	Tables []string `json:"tables"`
}

func GetCSVLine(productUUID uuid.UUID, csvpath string) (UUIDCSV, error) {
	isFirstRow := true
	headerMap := make(map[string]int)

	// Open csv file
	file, err := os.Open(csvpath)
	if err != nil {
		return UUIDCSV{}, err
	}
	defer file.Close()

	// Read csv values using csv.Reader
	csvReader := csv.NewReader(file)

	// Loop through lines
	for {
		rec, err := csvReader.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			return UUIDCSV{}, err
		}

		if isFirstRow {
			isFirstRow = false

			// Add mapping: Column/property name --> record index
			for i, v := range rec {
				headerMap[v] = i
			}

			// Skip next code
			continue
		}

		// Parse the uuid field into uuid type.
		recUUID, err := uuid.Parse(rec[headerMap["uuid"]])
		if err != nil {
			errMessage := fmt.Errorf("The UUID (%s) is invalid.", rec[headerMap["uuid"]])
			return UUIDCSV{}, errMessage
		}

		// If uuid matches return the line.
		if recUUID == productUUID {
			return UUIDCSV{recUUID.String(), strings.Split(rec[headerMap["tables"]], " ")}, nil
		}

	}

	// Return error if no uuid matches.
	return UUIDCSV{}, fmt.Errorf("Couldn't find the requested UUID %s, data not available.", productUUID.String())
}

func WKTToWFSFilter(wkt string) (SpatialFilter, error) {
	r := regexp.MustCompile(`((SRID=(?P<SRID>\d{4});)*(?P<GeometryType>[A-Za-z]*) *\(\((?P<Coordinates>.*)\)\))`)
	matches := r.FindStringSubmatch(wkt)

	if matches != nil {
		return SpatialFilter{
			matches[r.SubexpIndex("SRID")],
			matches[r.SubexpIndex("GeometryType")],
			matches[r.SubexpIndex("Coordinates")]}, nil
	}

	// Create the error message
	errMsg := fmt.Errorf("The provided (E)WKT (%s) is invalid, make sure that this is a polygon.", wkt)

	return SpatialFilter{}, errMsg
}

func VectorExtraction(c *gin.Context) {
	// Create URLParameters object.
	var urlparam RequestParameters

	// Get the url parameters.
	err := c.BindJSON(&urlparam)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get spatial filter info.
	//spatialf, err := WKTToWFSFilter(urlparam.Perimeter)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse the receivedd UUID, error if not parsable.
	productUUID, err := uuid.Parse(urlparam.Product)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Read the configuration csv and return the corresponding line.
	productTable, err := GetCSVLine(productUUID, "tests/extract.csv")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create the WFSRequestParameters objet
	var wfsrequests []url.URL
	wfsrequestparam := WFSRequestParameters{
		BaseURL:   "https://map.lausanne.ch/mapserv_proxy",
		OGCServer: "source for image/png",
		Service:   "wfs",
		Version:   "2.0.0",
		Request:   "GetFeature",
		TypeNames: productTable.Tables,
	}

	for _, value := range productTable.Tables {
		baseUrl, err := url.Parse(wfsrequestparam.BaseURL)
		if err != nil {
			fmt.Println("Malformed URL: ", err.Error())
			return
		}

		// Prepare Query Parameters
		params := url.Values{}
		params.Add("ogcserver", wfsrequestparam.OGCServer)
		params.Add("service", wfsrequestparam.Service)
		params.Add("version", wfsrequestparam.Version)
		params.Add("request", wfsrequestparam.Request)
		params.Add("ogcserver", wfsrequestparam.OGCServer)
		params.Add("typenames", value)

		// Add Query Parameters to the URL
		baseUrl.RawQuery = params.Encode()

		wfsrequests = append(wfsrequests, *baseUrl)
	}

	result, err := http.Get(wfsrequests[0].String())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	defer result.Body.Close()

	// Read GML data from the response body
	b, _ := io.ReadAll(result.Body)

	// Open new file and write data into gml format
	gmlFile := "tests/response.gml"
	err = os.WriteFile(gmlFile, b, 0644)

	err = godal.RegisterVector("GML")
	err = godal.RegisterVector("ESRI Shapefile")
	//gmlDriver, ok := godal.VectorDriver(gmlDriverName)

	// Open the GML file
	ds, err := godal.Open(gmlFile, godal.VectorOnly())
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer ds.Close()

	// Get the GML layer
	layers := ds.Layers()
	if len(layers) == 0 {
		log.Fatal("File does not contain any layers")
	}
	layer := layers[0]
	layerType := layer.Type()
	layerSrs := layer.SpatialRef()
	layerName := layer.Name()

	// Create a new Shapefile
	driver := godal.DriverName("ESRI Shapefile")

	shapefileDS, err := godal.CreateVector(driver, "tests/output.shp")
	if err != nil {
		log.Fatal("Can't create output dataset.")
	}
	defer shapefileDS.Close()

	// Get fields
	fields := layer.NextFeature().Fields()
	var fieldDefinitions []godal.CreateLayerOption
	for key, field := range fields {
		fmt.Printf("%s: %+v\n", key, field)
		fieldDefinitions = append(fieldDefinitions, godal.NewFieldDefinition(key, field.Type()))
		if err != nil {
			log.Fatal("Can't create fields.")
		}
	}

	// Create a new layer in the Shapefile
	shapefileLayer, err := shapefileDS.CreateLayer(layerName, layerSrs, layerType, fieldDefinitions...)
	if err != nil {
		log.Fatalf("Failed to set geometry: %v", err)
	}

	// Copy features from the GML layer to the Shapefile layer
	layer.ResetReading()
	for {
		feature := layer.NextFeature()
		if feature == nil {
			break
		}

		err = shapefileLayer.CreateFeature(feature)
		if err != nil {
			log.Fatal("Can't create feature.")
		}

		feature.Close()
	}
}
