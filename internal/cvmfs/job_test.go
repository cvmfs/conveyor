package cvmfs

import (
	"bytes"
	"strings"
	"testing"
)

const input = "What goes in must also come out"

func TestScriptSerialization(t *testing.T) {
	reader := strings.NewReader(input)

	packed, err := packScript(reader)
	if err != nil {
		t.Errorf("could not pack data. Err: %v\n", err)
	}

	var writer bytes.Buffer
	if err := unpackScript(packed, &writer); err != nil {
		t.Errorf("could not unpack data. Err: %v\n", err)
	}

	output := string(writer.Bytes())
	if input != output {
		t.Errorf(
			"Packing/Unpacking error. Input (%v): %v, Output (%v): %v\n",
			len(input), input, len(output), output)
	}
}
