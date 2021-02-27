package main

import (
	"io/ioutil"
	"os"
	"testing"
)

var themeDir = "./public/themes"
var imageNames = [...]string{"data01.png", "data02.png", "data03.png"}

func TestThemeImages(t *testing.T) {
	var err error

	_, err = os.Stat(themeDir)
	if err != nil {
		t.Fatal("themes file is not found : ", err)
	}

	themedirs, err := ioutil.ReadDir(themeDir)
	if err != nil {
		t.Fatal("themes file is not found : ", err)
	}

	for _, themedir := range themedirs {
		if themedir.IsDir() {
			t.Log("found theme : ", themedir.Name())
			files, err := ioutil.ReadDir(themeDir + "/" + themedir.Name())
			if err != nil {
				t.Log("theme image is not found. skipping,,, : ", err)
				continue
			}
			chkflg := [len(imageNames)]bool{}
			for _, file := range files {
				if !file.IsDir() {
					for i, v := range imageNames {
						if file.Name() == v {
							chkflg[i] = true
						}
					}
				}
			}
			for i, v := range chkflg {
				if !v {
					t.Error(imageNames[i] + " is not found in " + themedir.Name() + "theme")
				}
			}

		}
	}
}
