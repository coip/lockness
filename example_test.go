// Test and example for package lockness
package lockness_test

import (
	"fmt"
	"log"
	"os"

	"github.com/FATHOM5/lockness"
)

// ExampleLLRequest_Progress demonstrates a simple use of LLReq Progress function
func ExampleLLRequest_Progress() {

	testUser := "mark"
	llReq, syllabus := lockness.NewLLRequest("../llcfg.yml", "../modules.json")
	if llReq.Err != nil {
		log.Fatal(llReq.Err.Error())
	}
	if syllabus.Err != nil {
		log.Fatal(syllabus.Err.Error())
	}
	pd, err := llReq.Progress(testUser, syllabus)
	if err != nil {
		fmt.Printf("Error in Progress: %s", err)
		os.Exit(1)
	}
	fmt.Println(pd)
}
