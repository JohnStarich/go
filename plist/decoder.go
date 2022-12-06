package plist

import (
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"io"
	"log"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// ToJSON decodes 'r' containing an XML plist and returns the value as JSON-encoded bytes.
//
// For example, the following XML:
//
//	<array>
//		<integer>1</integer>
//		<string>foo</string>
//	</array>
//
// is returned as `[1, "foo"]`.
func ToJSON(r io.Reader) ([]byte, error) {
	value, err := newDecoder(r).toGo()
	if err != nil && err != io.EOF {
		return nil, err
	}
	return json.Marshal(value)
}

type decoder struct {
	xmlDecoder *xml.Decoder
	logger     *log.Logger
}

func newDecoder(r io.Reader) *decoder {
	return &decoder{
		xmlDecoder: xml.NewDecoder(r),
		logger:     log.New(io.Discard, "", 0),
	}
}

func (d *decoder) toGo() (value interface{}, err error) {
	_, _, err = d.decodeGo(d.xmlDecoder, 0)
	if err == nil || err == foundPListError {
		value, _, err = d.decodeGo(d.xmlDecoder, 0)
	}
	return value, err
}

var foundPListError = errors.New("found plist root")

func (d *decoder) decodeGo(decoder *xml.Decoder, depth int) (result interface{}, foundEndImmediately bool, err error) {
	elem, foundEnd, err := decodeStartElement(decoder)
	if foundEnd || err != nil {
		return nil, foundEnd, err
	}
	logPad := strings.Repeat("  ", depth)
	d.logger.Print(logPad, "start ", elem.Name.Local, "\n")
	defer func() {
		d.logger.Print(logPad, "end ", elem.Name.Local, ": ", result, "\n")
		if err != nil {
			d.logger.Println(" err: ", err)
		}
	}()

	switch elem.Name.Local {
	case "plist":
		return nil, false, foundPListError
	case "array":
		var elems []interface{}
		for {
			elem, foundEndImmediately, err := d.decodeGo(decoder, depth+1)
			if foundEndImmediately || err != nil {
				return elems, false, err
			}
			elems = append(elems, elem)
		}
	case "dict":
		elems := make(map[string]interface{})
		for {
			keyElem, foundEnd, err := decodeStartElement(decoder)
			if foundEnd || err != nil {
				return elems, false, err
			}
			var key string
			err = decoder.DecodeElement(&key, &keyElem)
			if err != nil {
				return elems, false, err
			}
			value, foundEndImmediately, err := d.decodeGo(decoder, depth+1)
			if foundEndImmediately || err != nil {
				return elems, false, err
			}
			elems[key] = value
		}
	case "true":
		return true, false, decoder.Skip()
	case "false":
		return false, false, decoder.Skip()
	case "integer":
		var value int64
		err := decoder.DecodeElement(&value, &elem)
		return value, false, err
	case "real":
		var value float64
		err := decoder.DecodeElement(&value, &elem)
		return value, false, err
	case "string":
		var value string
		err := decoder.DecodeElement(&value, &elem)
		return value, false, err
	case "data":
		var value string
		err := decoder.DecodeElement(&value, &elem)
		if err != nil {
			return nil, false, err
		}
		decoded, err := base64.StdEncoding.DecodeString(value)
		return decoded, false, err
	case "date":
		var value string
		err := decoder.DecodeElement(&value, &elem)
		if err != nil {
			return nil, false, err
		}
		parsed, err := time.Parse(time.RFC3339, value)
		return parsed, false, err
	default:
		return nil, false, errors.Errorf("unrecognized type: %s", elem.Name.Local)
	}
}

func decodeStartElement(decoder *xml.Decoder) (startElement xml.StartElement, foundEnd bool, err error) {
	for {
		var token xml.Token
		token, err = decoder.Token()
		if err != nil {
			return
		}
		switch token := token.(type) {
		case xml.StartElement:
			return token, false, nil
		case xml.EndElement:
			return xml.StartElement{}, true, nil
		}
	}
}
