package pdf

import (
	"fmt"
	"io"
	"strings"

	"github.com/go-pdf/fpdf"
)

// InvoiceGenerator represents an invoice generator.
type InvoiceGenerator struct {
	pdf *fpdf.Fpdf

	// Input flags
	InvoiceNo   string
	InvoiceDate string
	CompanyNo   string
	FromName    string
	FromAddress string
	FromContact string
	ToName      string
	ToAddress   string
	ToContact   string
}

// SubscriptionInfo represents the information used to generate the invoice
type SubscriptionInfo struct {
	ProductDescription string
	Quantity           int
	UnitPrice          float64
	Price              float64
	SubTotal           float64
	Tax                int
	TaxAmount          float64
	GrandTotal         float64
	Currency           string
	CurrencySymbol     string
}

// NewInvoiceGenerator creates a new instance of InvoiceGenerator.
func NewInvoiceGenerator() *InvoiceGenerator {
	return &InvoiceGenerator{
		pdf: fpdf.New("P", "mm", "A4", ""),
	}
}

// GenerateInvoice generates the invoice.
func (ig *InvoiceGenerator) GenerateInvoice(data SubscriptionInfo, w io.Writer, logoImage, logoImageType string) error {
	marginX := 10.0
	marginY := 20.0
	gapY := 2.0
	ig.pdf.SetMargins(marginX, marginY, marginX)
	ig.pdf.AddPage()
	pageW, _ := ig.pdf.GetPageSize()
	safeAreaW := pageW - 2*marginX

	ig.pdf.ImageOptions(logoImage, 0, 0, 65, 25, false, fpdf.ImageOptions{ImageType: logoImageType, ReadDpi: true}, 0, "")
	ig.pdf.SetFont("Arial", "B", 16)
	_, lineHeight := ig.pdf.GetFontSize()
	currentY := ig.pdf.GetY() + lineHeight + gapY
	ig.pdf.SetXY(marginX, currentY)
	ig.pdf.Cell(40, 10, ig.FromName)

	if ig.CompanyNo != "" {
		ig.pdf.SetFont("Arial", "BI", 12)
		_, lineHeight = ig.pdf.GetFontSize()
		ig.pdf.SetXY(marginX, ig.pdf.GetY()+lineHeight+gapY)
		ig.pdf.Cell(40, 10, fmt.Sprintf("Company No : %v", ig.CompanyNo))
	}

	leftY := ig.pdf.GetY() + lineHeight + gapY
	// Build invoice word on right
	ig.pdf.SetFont("Arial", "B", 32)
	_, lineHeight = ig.pdf.GetFontSize()
	ig.pdf.SetXY(130, currentY-lineHeight)
	ig.pdf.Cell(100, 40, "INVOICE")

	newY := leftY
	if (ig.pdf.GetY() + gapY) > newY {
		newY = ig.pdf.GetY() + gapY
	}

	newY += 10.0 // Add margin

	ig.pdf.SetXY(marginX, newY)
	ig.pdf.SetFont("Arial", "", 12)
	_, lineHeight = ig.pdf.GetFontSize()
	lineBreak := lineHeight + float64(1)

	// Left hand info
	splittedFromAddress := ig.breakAddress(ig.FromAddress)
	for _, add := range splittedFromAddress {
		ig.pdf.Cell(safeAreaW/2, lineHeight, add)
		ig.pdf.Ln(lineBreak)
	}
	ig.pdf.SetFontStyle("I")
	ig.pdf.Cell(safeAreaW/2, lineHeight, fmt.Sprintf("Tel: %s", ig.FromContact))
	ig.pdf.Ln(lineBreak)
	ig.pdf.Ln(lineBreak)
	ig.pdf.Ln(lineBreak)

	ig.pdf.SetFontStyle("B")
	ig.pdf.Cell(safeAreaW/2, lineHeight, "Bill To:")
	ig.pdf.Line(marginX, ig.pdf.GetY()+lineHeight, marginX+safeAreaW/2, ig.pdf.GetY()+lineHeight)
	ig.pdf.Ln(lineBreak)
	ig.pdf.Cell(safeAreaW/2, lineHeight, ig.ToName)
	ig.pdf.SetFontStyle("")
	ig.pdf.Ln(lineBreak)
	splittedToAddress := ig.breakAddress(ig.ToAddress)
	for _, add := range splittedToAddress {
		ig.pdf.Cell(safeAreaW/2, lineHeight, add)
		ig.pdf.Ln(lineBreak)
	}
	ig.pdf.SetFontStyle("I")
	ig.pdf.Cell(safeAreaW/2, lineHeight, fmt.Sprintf("Tel: %s", ig.ToContact))

	endOfInvoiceDetailY := ig.pdf.GetY() + lineHeight
	ig.pdf.SetFontStyle("")

	// Right hand side info, invoice no & invoice date
	invoiceDetailW := float64(30)
	ig.pdf.SetXY(safeAreaW/2+30, newY)
	ig.pdf.Cell(invoiceDetailW, lineHeight, "Invoice No.:")
	ig.pdf.Cell(invoiceDetailW, lineHeight, ig.InvoiceNo)
	ig.pdf.Ln(lineBreak)
	ig.pdf.SetX(safeAreaW/2 + 30)
	ig.pdf.Cell(invoiceDetailW, lineHeight, "Invoice Date:")
	ig.pdf.Cell(invoiceDetailW, lineHeight, ig.InvoiceDate)
	ig.pdf.Ln(lineBreak)

	// Draw the table
	ig.drawTable(data, marginX, endOfInvoiceDetailY, 10.0)

	ig.pdf.SetFontStyle("")
	ig.pdf.Ln(lineBreak)
	ig.pdf.Cell(safeAreaW, lineHeight, "Note: The tax invoice is computer generated and no signature is required.")

	return ig.pdf.Output(w)
}

// SetFromAddress sets the 'From' address.
func (ig *InvoiceGenerator) SetFromAddress(address string) {
	ig.FromAddress = address
}

// SetToAddress sets the 'To' address.
func (ig *InvoiceGenerator) SetToAddress(address string) {
	ig.ToAddress = address
}

// breakAddress breaks the address string into lines for formatting.
func (ig *InvoiceGenerator) breakAddress(input string) []string {
	var address []string
	splitted := strings.Split(input, ",")
	prevAddress := ""
	for _, add := range splitted {
		if len(add) < 10 {
			prevAddress = add
			continue
		}
		currentAdd := strings.TrimSpace(add)
		if prevAddress != "" {
			currentAdd = prevAddress + ", " + currentAdd
		}
		address = append(address, currentAdd)
		prevAddress = ""
	}

	return address
}

// drawTable draws the table with invoice data.
func (ig *InvoiceGenerator) drawTable(data SubscriptionInfo, marginX, startY, lineHeight float64) {
	ig.pdf.SetXY(marginX, startY+10.0)
	const colNumber = 5
	header := [colNumber]string{"No", "Description", "Quantity", fmt.Sprintf("Unit Price (%s)", data.CurrencySymbol), fmt.Sprintf("Price (%s)", data.CurrencySymbol)}
	colWidth := [colNumber]float64{10.0, 75.0, 25.0, 40.0, 40.0}

	// Headers
	ig.pdf.SetFontStyle("B")
	ig.pdf.SetFillColor(200, 200, 200)
	for colJ := 0; colJ < colNumber; colJ++ {
		ig.pdf.CellFormat(colWidth[colJ], lineHeight, header[colJ], "1", 0, "CM", true, 0, "")
	}

	ig.pdf.Ln(-1)
	ig.pdf.SetFillColor(255, 255, 255)

	// Table data
	ig.pdf.SetFontStyle("")

	ig.pdf.CellFormat(colWidth[0], lineHeight, fmt.Sprintf("%d", 1), "1", 0, "CM", true, 0, "")
	ig.pdf.CellFormat(colWidth[1], lineHeight, data.ProductDescription, "1", 0, "LM", true, 0, "")
	ig.pdf.CellFormat(colWidth[2], lineHeight, fmt.Sprintf("%d", data.Quantity), "1", 0, "CM", true, 0, "")
	ig.pdf.CellFormat(colWidth[3], lineHeight, fmt.Sprintf("%.2f", data.UnitPrice), "1", 0, "CM", true, 0, "")
	ig.pdf.CellFormat(colWidth[4], lineHeight, fmt.Sprintf("%.2f", data.Price), "1", 0, "CM", true, 0, "")
	ig.pdf.Ln(-1)

	ig.pdf.SetFontStyle("B")
	leftIndent := 0.0
	for i := 0; i < 3; i++ {
		leftIndent += colWidth[i]
	}
	ig.pdf.SetX(marginX + leftIndent)
	ig.pdf.CellFormat(colWidth[3], lineHeight, "Subtotal", "1", 0, "CM", true, 0, "")
	ig.pdf.CellFormat(colWidth[4], lineHeight, fmt.Sprintf("%.2f", data.SubTotal), "1", 0, "CM", true, 0, "")
	ig.pdf.Ln(-1)

	ig.pdf.SetX(marginX + leftIndent)
	ig.pdf.CellFormat(colWidth[3], lineHeight, "Tax Amount", "1", 0, "CM", true, 0, "")
	ig.pdf.CellFormat(colWidth[4], lineHeight, fmt.Sprintf("%.2f", data.TaxAmount), "1", 0, "CM", true, 0, "")
	ig.pdf.Ln(-1)

	ig.pdf.SetX(marginX + leftIndent)
	ig.pdf.CellFormat(colWidth[3], lineHeight, "Grand total", "1", 0, "CM", true, 0, "")
	ig.pdf.CellFormat(colWidth[4], lineHeight, fmt.Sprintf("%.2f", data.GrandTotal), "1", 0, "CM", true, 0, "")
	ig.pdf.Ln(-1)
}

// SetInvoiceNo sets the invoice number.
func (ig *InvoiceGenerator) SetInvoiceNo(invoiceNo string) {
	ig.InvoiceNo = invoiceNo
}

// SetInvoiceDate sets the invoice date.
func (ig *InvoiceGenerator) SetInvoiceDate(invoiceDate string) {
	ig.InvoiceDate = invoiceDate
}

// SetCompanyNo sets the company number.
func (ig *InvoiceGenerator) SetCompanyNo(companyNo string) {
	ig.CompanyNo = companyNo
}

// SetFromName sets the 'From' name.
func (ig *InvoiceGenerator) SetFromName(fromName string) {
	ig.FromName = fromName
}

// SetFromContact sets the 'From' contact.
func (ig *InvoiceGenerator) SetFromContact(contact string) {
	ig.FromContact = contact
}

// SetToName sets the 'To' name.
func (ig *InvoiceGenerator) SetToName(toName string) {
	ig.ToName = toName
}

// SetToContact sets the 'To' contact.
func (ig *InvoiceGenerator) SetToContact(contact string) {
	ig.ToContact = contact
}
