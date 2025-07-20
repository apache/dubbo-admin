/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cluster

import (
	"fmt"
	"io"
	"strings"
)

type writerPrinter struct {
	writer io.Writer
}

type Printer interface {
	Printf(string, ...any)
	Println(string)
}

func NewPrinterForWriter(w io.Writer) Printer {
	return &writerPrinter{writer: w}
}

func (w writerPrinter) Printf(format string, a ...any) {
	_, _ = fmt.Fprintf(w.writer, format, a...)
}

func (w writerPrinter) Println(s string) {
	_, _ = fmt.Fprintln(w.writer, s)
}

// OptionDeterminate waits for a user to confirm with the supplied message.
func OptionDeterminate(msg string, writer io.Writer) bool {
	for {
		_, _ = fmt.Fprintf(writer, "%s ", msg)
		var resp string
		_, err := fmt.Scanln(&resp)
		if err != nil {
			return false
		}
		switch strings.ToUpper(resp) {
		case "Y", "YES", "y", "yes":
			return true
		case "N", "NO", "n", "no":
			return false
		}
	}
}
