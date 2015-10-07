// Author  Raido Pahtma
// License MIT

package reportreceiver

import "fmt"
import "os"

type ReportFileWriter struct {
	outfile string
}

func NewReportFileWriter(outfile string) (*ReportFileWriter, error) {
	rfw := new(ReportFileWriter)
	rfw.outfile = outfile
	return rfw, nil
}

// Append a report to the log.
func (self *ReportFileWriter) Append(rd *Report) error {
	var out *os.File
	var err error
	out, err = os.OpenFile(self.outfile, os.O_CREATE|os.O_EXCL|os.O_WRONLY|os.O_APPEND, 0660)
	if err == nil {
		out.WriteString(fmt.Sprintf("######,%s\n", rd.StorageStringHeader()))
	} else {
		out, err = os.OpenFile(self.outfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
		if err != nil {
			return err
		}
	}
	defer out.Close()

	out.WriteString(fmt.Sprintf("REPORT,%s\n", rd.StorageString()))

	return nil
}
