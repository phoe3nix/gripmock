package stub

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"reflect"
	"regexp"
	"sync"

	"github.com/lithammer/fuzzysearch/fuzzy"
)

var mx = sync.Mutex{}

// below represent map[servicename][methodname][]expectations
type stubMapping map[string]map[string][]storage

var stubStorage = stubMapping{}

type storage struct {
	Input  Input
	Meta   Meta
	Output Output
}

func storeStub(stub *Stub) error {
	mx.Lock()
	defer mx.Unlock()

	strg := storage{
		Input:  stub.Input,
		Meta:	stub.Meta,
		Output: stub.Output,
	}
	if stubStorage[stub.Service] == nil {
		stubStorage[stub.Service] = make(map[string][]storage)
	}
	stubStorage[stub.Service][stub.Method] = append(stubStorage[stub.Service][stub.Method], strg)
	return nil
}

func allStub() stubMapping {
	mx.Lock()
	defer mx.Unlock()
	return stubStorage
}

type closeMatch struct {
	rule   string
	expect map[string]interface{}
}

func findStub(stub *findStubPayload) (*Output, error) {
	mx.Lock()
	defer mx.Unlock()
	if _, ok := stubStorage[stub.Service]; !ok {
		return nil, fmt.Errorf("Can't find stub for Service: %s", stub.Service)
	}

	if _, ok := stubStorage[stub.Service][stub.Method]; !ok {
		return nil, fmt.Errorf("Can't find stub for Service:%s and Method:%s", stub.Service, stub.Method)
	}

	stubs := stubStorage[stub.Service][stub.Method]
	if len(stubs) == 0 {
		return nil, fmt.Errorf("Stub for Service:%s and Method:%s is empty", stub.Service, stub.Method)
	}

	closestMatch := []closeMatch{}
	for _, stubrange := range stubs {
		if expect := stubrange.Input.Equals; expect != nil {
			closestMatch = append(closestMatch, closeMatch{"equals", expect})
			if equals(stub.Data, expect) {
				if !metaEquals(stub.Meta, stubrange.Meta) {
					continue
				}
				return &stubrange.Output, nil
			}
		}

		if expect := stubrange.Input.Contains; expect != nil {
			closestMatch = append(closestMatch, closeMatch{"contains", expect})
			if contains(stubrange.Input.Contains, stub.Data) {
				if !metaEquals(stub.Meta, stubrange.Meta) {
					continue
				}
				return &stubrange.Output, nil
			}
		}

		if expect := stubrange.Input.Matches; expect != nil {
			closestMatch = append(closestMatch, closeMatch{"matches", expect})
			if matches(stubrange.Input.Matches, stub.Data) {
				if !metaEquals(stub.Meta, stubrange.Meta) {
					continue
				}
				return &stubrange.Output, nil
			}
		}
	}

	return nil, stubNotFoundError(stub, closestMatch)
}

func stubNotFoundError(stub *findStubPayload, closestMatches []closeMatch) error {
	template := fmt.Sprintf("Can't find stub \n\nService: %s \n\nMethod: %s \n\nInput\n\n", stub.Service, stub.Method)
	expectString := renderFieldAsString(stub.Data)
	template += expectString

	if len(closestMatches) == 0 {
		return fmt.Errorf(template)
	}

	highestRank := struct {
		rank  float32
		match closeMatch
	}{0, closeMatch{}}
	for _, closeMatchValue := range closestMatches {
		rank := rankMatch(expectString, closeMatchValue.expect)

		// the higher the better
		if rank > highestRank.rank {
			highestRank.rank = rank
			highestRank.match = closeMatchValue
		}
	}

	var closestMatch closeMatch
	if highestRank.rank == 0 {
		closestMatch = closestMatches[0]
	} else {
		closestMatch = highestRank.match
	}

	closestMatchString := renderFieldAsString(closestMatch.expect)
	template += fmt.Sprintf("\n\nClosest Match \n\n%s:%s", closestMatch.rule, closestMatchString)

	return fmt.Errorf(template)
}

// we made our own simple ranking logic
// count the matches field_name and value then compare it with total field names and values
// the higher the better
func rankMatch(expect string, closeMatch map[string]interface{}) float32 {
	occurence := 0
	for key, value := range closeMatch {
		if fuzzy.Match(key+":", expect) {
			occurence++
		}

		if fuzzy.Match(fmt.Sprint(value), expect) {
			occurence++
		}
	}

	if occurence == 0 {
		return 0
	}
	totalFields := len(closeMatch) * 2
	return float32(occurence) / float32(totalFields)
}

func renderFieldAsString(fields map[string]interface{}) string {
	template := "{\n"
	for key, val := range fields {
		template += fmt.Sprintf("\t%s: %v\n", key, val)
	}
	template += "}"
	return template
}

func equals(input1, input2 map[string]interface{}) bool {
	return reflect.DeepEqual(input1, input2)
}

func metaEquals(input1, input2 Meta) bool {
	return reflect.DeepEqual(input1, input2)
}

func contains(expect, actual map[string]interface{}) bool {
	for key, val := range expect {
		actualvalue, ok := actual[key]
		if !ok {
			return ok
		}

		if !reflect.DeepEqual(val, actualvalue) {
			return false
		}
	}
	return true
}

func matches(expect, actual map[string]interface{}) bool {
	for keyExpect, valueExpect := range expect {
		valueExpectString, ok := valueExpect.(string)
		if !ok {
			return false
		}
		actualvalue, ok := actual[keyExpect].(string)
		if !ok {
			return false
		}

		match, err := regexp.Match(valueExpectString, []byte(actualvalue))
		if err != nil {
			log.Println("Error on matching regex %s with %s error:%v", valueExpectString, actualvalue, err)
		}

		if !match {
			return false
		}
	}
	return true
}

func clearStorage(meta *Meta) {
	mx.Lock()
	defer mx.Unlock()

	// newStubStorage := stubStorage
	newStubStorage := stubMapping{}

	for serviceKey, serviceValue := range stubStorage {
		for methodKey, _ := range serviceValue {
			storageStubs := stubStorage[serviceKey][methodKey]
			for _, storageStub := range storageStubs {
				// if metaEquals(storageStub.Meta, *meta) {
					// log.Printf("%v\n", index)
					// log.Printf("%v\n", storageStubs)
					// storageAfterRemove := removeIndex(newStubStorage[serviceKey][methodKey], index)
					// newStubStorage[serviceKey][methodKey] = storageAfterRemove
					// removeIndex(newStubStorage[serviceKey][methodKey], index)
				// }
				if !metaEquals(storageStub.Meta, *meta) {
					if newStubStorage[serviceKey] == nil {
						newStubStorage[serviceKey] = make(map[string][]storage)
					}
					newStubStorage[serviceKey][methodKey] = append(newStubStorage[serviceKey][methodKey], storageStub)
				}
			}
		}
	}

	stubStorage = newStubStorage
}

func removeIndex(s []storage, index int) []storage {
    return append(s[:index], s[index+1:]...)
}

func readStubFromFile(path string) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Printf("Can't read stub from %s. %v\n", path, err)
		return
	}

	for _, file := range files {
		byt, err := ioutil.ReadFile(path + "/" + file.Name())
		if err != nil {
			log.Printf("Error when reading file %s. %v. skipping...", file.Name(), err)
			continue
		}

		stub := new(Stub)
		err = json.Unmarshal(byt, stub)
		if err != nil {
			log.Printf("Error when reading file %s. %v. skipping...", file.Name(), err)
			continue
		}

		storeStub(stub)
	}
}
