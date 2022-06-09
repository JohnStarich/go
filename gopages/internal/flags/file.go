package flags

import "io/ioutil"

// FilePathContents is a flag that reads a file by the passed in path and contains its contents
type FilePathContents struct {
	contents []byte
}

// Contents returns the contents of this flag's file
func (f *FilePathContents) Contents() []byte {
	return f.contents
}

func (f *FilePathContents) String() string {
	if f == nil || f.contents == nil {
		return ""
	}
	return string(f.contents)
}

// Set implements flag.Value
func (f *FilePathContents) Set(s string) error {
	contents, err := ioutil.ReadFile(s)
	if err != nil {
		return err
	}
	f.contents = contents
	return nil
}
