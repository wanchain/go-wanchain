// based on http://sabhiram.com/go/2015/01/21/golang_trace_fns_part_1.html
package functrace

import (
	"fmt"
	"regexp"
	"runtime"

	"github.com/wanchain/go-wanchain/log"
)

// Trace Functions
func Enter(str ...string) {
	// Skip this function, and fetch the PC and file for its parent
	pc, _, _, _ := runtime.Caller(1)
	// Retrieve a Function object this functions parent
	functionObject := runtime.FuncForPC(pc)
	// Regex to extract just the function name (and not the module path)
	extractFnName := regexp.MustCompile(`^.*\.(.*)$`)
	fnName := extractFnName.ReplaceAllString(functionObject.Name(), "$1")

	var smsg string
	if len(str) > 0 {
		smsg = str[0]
	}
	log.Debug(fmt.Sprintf(">> Entering %s(%s)", fnName, smsg))
}

func Exit() {
	// Skip this function, and fetch the PC and file for its parent
	pc, _, _, _ := runtime.Caller(1)
	// Retrieve a Function object this functions parent
	functionObject := runtime.FuncForPC(pc)
	// Regex to extract just the function name (and not the module path)
	extractFnName := regexp.MustCompile(`^.*\.(.*)$`)
	fnName := extractFnName.ReplaceAllString(functionObject.Name(), "$1")
	log.Debug(fmt.Sprintf("<< Exiting  %s()", fnName))
}
