package service

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type xlsxCell struct {
	Text   string
	Number *float64
	Style  int
}

type xlsxColumn struct {
	Header string
	Width  float64
}

type xlsxSheet struct {
	Name    string
	Columns []xlsxColumn
	Rows    [][]xlsxCell
}

func textCell(value string) xlsxCell {
	return xlsxCell{Text: value}
}

func integerCell(value int64) xlsxCell {
	number := float64(value)
	return xlsxCell{Number: &number, Style: 2}
}

func amountCell(value float64) xlsxCell {
	return xlsxCell{Number: &value, Style: 3}
}

func totalTextCell(value string) xlsxCell {
	return xlsxCell{Text: value, Style: 4}
}

func totalIntegerCell(value int64) xlsxCell {
	number := float64(value)
	return xlsxCell{Number: &number, Style: 4}
}

func totalAmountCell(value float64) xlsxCell {
	return xlsxCell{Number: &value, Style: 5}
}

func buildXLSX(sheets []xlsxSheet) ([]byte, error) {
	if len(sheets) == 0 {
		return nil, fmt.Errorf("at least one worksheet is required")
	}

	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	files := map[string]string{
		"[Content_Types].xml":        contentTypesXML(len(sheets)),
		"_rels/.rels":                packageRelationshipsXML,
		"docProps/app.xml":           appPropertiesXML(len(sheets)),
		"docProps/core.xml":          corePropertiesXML(),
		"xl/workbook.xml":            workbookXML(sheets),
		"xl/_rels/workbook.xml.rels": workbookRelationshipsXML(len(sheets)),
		"xl/styles.xml":              stylesXML,
	}
	for path, content := range files {
		file, err := writer.Create(path)
		if err != nil {
			_ = writer.Close()
			return nil, err
		}
		if _, err := file.Write([]byte(content)); err != nil {
			_ = writer.Close()
			return nil, err
		}
	}

	for index, sheet := range sheets {
		file, err := writer.Create(fmt.Sprintf("xl/worksheets/sheet%d.xml", index+1))
		if err != nil {
			_ = writer.Close()
			return nil, err
		}
		if err := writeWorksheetXML(file, sheet); err != nil {
			_ = writer.Close()
			return nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func writeWorksheetXML(writer interface{ Write([]byte) (int, error) }, sheet xlsxSheet) error {
	var content strings.Builder
	content.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	content.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	lastColumn := columnName(len(sheet.Columns))
	lastRow := len(sheet.Rows) + 1
	content.WriteString(`<dimension ref="A1:`)
	content.WriteString(lastColumn)
	content.WriteString(strconv.Itoa(lastRow))
	content.WriteString(`"/><sheetViews><sheetView workbookViewId="0"><pane ySplit="1" topLeftCell="A2" activePane="bottomLeft" state="frozen"/></sheetView></sheetViews>`)
	content.WriteString(`<cols>`)
	for index, column := range sheet.Columns {
		content.WriteString(`<col min="`)
		content.WriteString(strconv.Itoa(index + 1))
		content.WriteString(`" max="`)
		content.WriteString(strconv.Itoa(index + 1))
		content.WriteString(`" width="`)
		content.WriteString(strconv.FormatFloat(column.Width, 'f', 1, 64))
		content.WriteString(`" customWidth="1"/>`)
	}
	content.WriteString(`</cols><sheetData>`)

	content.WriteString(`<row r="1">`)
	for index, column := range sheet.Columns {
		appendStringCell(&content, columnName(index+1)+"1", column.Header, 1)
	}
	content.WriteString(`</row>`)
	for rowIndex, row := range sheet.Rows {
		excelRow := rowIndex + 2
		content.WriteString(`<row r="`)
		content.WriteString(strconv.Itoa(excelRow))
		content.WriteString(`">`)
		for columnIndex, cell := range row {
			if columnIndex >= len(sheet.Columns) {
				break
			}
			reference := columnName(columnIndex+1) + strconv.Itoa(excelRow)
			if cell.Number != nil {
				appendNumberCell(&content, reference, *cell.Number, cell.Style)
				continue
			}
			appendStringCell(&content, reference, cell.Text, cell.Style)
		}
		content.WriteString(`</row>`)
	}
	content.WriteString(`</sheetData>`)
	if len(sheet.Columns) > 0 {
		content.WriteString(`<autoFilter ref="A1:`)
		content.WriteString(lastColumn)
		content.WriteString(strconv.Itoa(lastRow))
		content.WriteString(`"/>`)
	}
	content.WriteString(`</worksheet>`)
	_, err := writer.Write([]byte(content.String()))
	return err
}

func appendStringCell(builder *strings.Builder, reference string, value string, style int) {
	value = sanitizeXLSXText(value)
	builder.WriteString(`<c r="`)
	builder.WriteString(reference)
	builder.WriteString(`" t="inlineStr"`)
	if style > 0 {
		builder.WriteString(` s="`)
		builder.WriteString(strconv.Itoa(style))
		builder.WriteString(`"`)
	}
	builder.WriteString(`><is><t xml:space="preserve">`)
	var escaped bytes.Buffer
	_ = xml.EscapeText(&escaped, []byte(value))
	builder.Write(escaped.Bytes())
	builder.WriteString(`</t></is></c>`)
}

func sanitizeXLSXText(value string) string {
	value = strings.Map(func(char rune) rune {
		if char == '\t' || char == '\n' || char == '\r' || char >= 0x20 {
			return char
		}
		return ' '
	}, value)
	runes := []rune(value)
	if len(runes) > 32767 {
		return string(runes[:32767])
	}
	return value
}

func appendNumberCell(builder *strings.Builder, reference string, value float64, style int) {
	builder.WriteString(`<c r="`)
	builder.WriteString(reference)
	builder.WriteString(`"`)
	if style > 0 {
		builder.WriteString(` s="`)
		builder.WriteString(strconv.Itoa(style))
		builder.WriteString(`"`)
	}
	builder.WriteString(`><v>`)
	builder.WriteString(strconv.FormatFloat(value, 'f', -1, 64))
	builder.WriteString(`</v></c>`)
}

func columnName(index int) string {
	if index <= 0 {
		return "A"
	}
	var name string
	for index > 0 {
		index--
		name = string(rune('A'+index%26)) + name
		index /= 26
	}
	return name
}

func contentTypesXML(sheetCount int) string {
	var content strings.Builder
	content.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/><Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/><Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/>`)
	for index := 1; index <= sheetCount; index++ {
		content.WriteString(`<Override PartName="/xl/worksheets/sheet`)
		content.WriteString(strconv.Itoa(index))
		content.WriteString(`.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`)
	}
	content.WriteString(`</Types>`)
	return content.String()
}

func workbookXML(sheets []xlsxSheet) string {
	var content strings.Builder
	content.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>`)
	for index, sheet := range sheets {
		content.WriteString(`<sheet name="`)
		var escaped bytes.Buffer
		_ = xml.EscapeText(&escaped, []byte(sheet.Name))
		content.Write(escaped.Bytes())
		content.WriteString(`" sheetId="`)
		content.WriteString(strconv.Itoa(index + 1))
		content.WriteString(`" r:id="rId`)
		content.WriteString(strconv.Itoa(index + 1))
		content.WriteString(`"/>`)
	}
	content.WriteString(`</sheets></workbook>`)
	return content.String()
}

func workbookRelationshipsXML(sheetCount int) string {
	var content strings.Builder
	content.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	for index := 1; index <= sheetCount; index++ {
		content.WriteString(`<Relationship Id="rId`)
		content.WriteString(strconv.Itoa(index))
		content.WriteString(`" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet`)
		content.WriteString(strconv.Itoa(index))
		content.WriteString(`.xml"/>`)
	}
	content.WriteString(`<Relationship Id="rId`)
	content.WriteString(strconv.Itoa(sheetCount + 1))
	content.WriteString(`" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>`)
	return content.String()
}

func appPropertiesXML(sheetCount int) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties" xmlns:vt="http://schemas.openxmlformats.org/officeDocument/2006/docPropsVTypes"><Application>new-api</Application><DocSecurity>0</DocSecurity><ScaleCrop>false</ScaleCrop><HeadingPairs><vt:vector size="2" baseType="variant"><vt:variant><vt:lpstr>Worksheets</vt:lpstr></vt:variant><vt:variant><vt:i4>%d</vt:i4></vt:variant></vt:vector></HeadingPairs></Properties>`, sheetCount)
}

func corePropertiesXML() string {
	createdAt := time.Now().UTC().Format(time.RFC3339)
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><dc:creator>new-api</dc:creator><cp:lastModifiedBy>new-api</cp:lastModifiedBy><dcterms:created xsi:type="dcterms:W3CDTF">%s</dcterms:created><dcterms:modified xsi:type="dcterms:W3CDTF">%s</dcterms:modified></cp:coreProperties>`, createdAt, createdAt)
}

const packageRelationshipsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/><Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/></Relationships>`

const stylesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><numFmts count="1"><numFmt numFmtId="164" formatCode="0.000000"/></numFmts><fonts count="2"><font><sz val="11"/><name val="Calibri"/></font><font><b/><sz val="11"/><color rgb="FFFFFFFF"/><name val="Calibri"/></font></fonts><fills count="3"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill><fill><patternFill patternType="solid"><fgColor rgb="FF4472C4"/><bgColor indexed="64"/></patternFill></fill></fills><borders count="2"><border><left/><right/><top/><bottom/><diagonal/></border><border><left style="thin"><color rgb="FFD9E2F3"/></left><right style="thin"><color rgb="FFD9E2F3"/></right><top style="thin"><color rgb="FFD9E2F3"/></top><bottom style="thin"><color rgb="FFD9E2F3"/></bottom><diagonal/></border></borders><cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs><cellXfs count="6"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/><xf numFmtId="0" fontId="1" fillId="2" borderId="1" xfId="0" applyAlignment="1"><alignment horizontal="center"/></xf><xf numFmtId="1" fontId="0" fillId="0" borderId="1" xfId="0"/><xf numFmtId="164" fontId="0" fillId="0" borderId="1" xfId="0" applyNumberFormat="1"/><xf numFmtId="0" fontId="1" fillId="2" borderId="1" xfId="0"/><xf numFmtId="164" fontId="1" fillId="2" borderId="1" xfId="0" applyNumberFormat="1"/></cellXfs><cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles></styleSheet>`
