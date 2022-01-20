package flags

import "io/ioutil"

type FilePathContents struct {
	contents []byte
}

func (f *FilePathContents) Contents() []byte {
	return f.contents
}

func (f *FilePathContents) String() string {
	if f == nil || f.contents == nil {
		return ""
	}
	return string(f.contents)
}

func (f *FilePathContents) Set(s string) error {
	contents, err := ioutil.ReadFile(s)
	if err != nil {
		return err
	}
	f.contents = contents
	return nil
}
