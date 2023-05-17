package extraction

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type URLParameters struct {
	Product    string `json:"product" binding:"required"`
	Perimeter  string `json:"perimeter" binding:"required"`
	Parameters string `json:"parameters" binding:"required"`
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
			return UUIDCSV{recUUID.String(), strings.Split(rec[headerMap["tables"]], ",")}, nil
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
	// Create URLParameters object
	var urlparam URLParameters

	// Get the url parameters
	err := c.BindJSON(&urlparam)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	spatialf, err := WKTToWFSFilter(urlparam.Perimeter)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, spatialf)

	/*perimeterStr := c.Query("perimeter")

	// Parse the perimeter value as a geometry
	perimeter, err := gdal.CreateFromWKT(perimeterStr, gdal.CreateSpatialReference("EPSG:4326"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid perimeter value.")
		return
	}

	// Open the vector layer using gdal
	dataset := gdal.OpenDataSource("data/test-points.shp", 0)
	if err != nil {
		c.String(http.StatusInternalServerError, "Could not open vector layer.")
		return
	}
	defer dataset.Destroy()

	// Get the layer from the dataset
	layer := dataset.LayerByIndex(0)

	// Create a spatial filter to select features based on the perimeter geometry
	layer.SetSpatialFilter(perimeter)

	// Create a feature array to hold the selected features

	features := make([]string, 0)

	// Loop over the features in the layer and add the selected ones to the feature array
	//layer.ResetReading()
	feature := layer.NextFeature()
	for feature != nil {

		// Fetch Geometry of feature
		geometry := feature.Geometry()

		// Convert Geomtry to WKT
		geometryWKT, err := geometry.ToWKT()

		if err != nil {
			c.String(http.StatusInternalServerError, "Could not get geometry.")
			return
		}

		// Add the feature to the feature array
		features = append(features, geometryWKT)

		// Get the next feature
		feature = layer.NextFeature()
	}

	// Print the number of selected features
	fmt.Printf("Selected %d features.\n", len(features))

	// Return the selected features as JSON
	c.JSON(http.StatusOK, gin.H{
		"features": features,
	})*/
}
