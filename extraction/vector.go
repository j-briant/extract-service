package extraction

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lukeroth/gdal"
)

func VectorExtraction(c *gin.Context) {
	// Get the value of the "perimeter" parameter
	perimeterStr := c.Query("perimeter")

	// Parse the perimeter value as a geometry
	perimeter, err := gdal.CreateFromWKT(perimeterStr, gdal.CreateSpatialReference("EPSG:4326"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid perimeter value.")
		return
	}

	// Open the vector layer using gdal
	dataset := gdal.OpenDataSource("data/test-points.shp", 0)
	/*if err != nil {
			c.String(http.StatusInternalServerError, "Could not open vector layer.")
			return
	}*/
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
	})
}
