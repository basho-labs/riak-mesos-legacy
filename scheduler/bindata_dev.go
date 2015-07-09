// +build !rel

package framework

import (
	"fmt"
	"io/ioutil"
)

// Asset reads the file at the abs path given
func Asset(name string) ([]byte, error) {
	dat, err := ioutil.ReadFile(name)

	if err != nil {
		return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
	}

	return dat, nil
}
