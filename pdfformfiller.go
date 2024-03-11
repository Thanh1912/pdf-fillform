package main

import (
	"bytes"
	"fmt"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/validate"
	"github.com/pkg/errors"
)

type FormField struct {
	ObjectId int
	Name     string
	Rect     *types.Rectangle
	Type     string
	Dict     types.Dict
}

type PDFFormFiller struct {
	ctx            *model.Context
	fieldMapById   map[int]*FormField
	fieldMapByName map[string]*FormField
}
type Form map[string]interface{}

func NewPDFFormFiller(template []byte) (r *PDFFormFiller, err error) {
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed

	ctx, err := pdfcpu.Read(bytes.NewReader(template), conf)
	if err != nil {
		return nil, err
	}

	if err = validate.XRefTable(ctx.XRefTable); err != nil {
		return nil, err
	}

	render := &PDFFormFiller{
		ctx:            ctx,
		fieldMapById:   make(map[int]*FormField),
		fieldMapByName: make(map[string]*FormField),
	}

	err = render.extractFormFields()
	if err != nil {
		return nil, err
	}
	return render, nil
}

func (r *PDFFormFiller) Fill(form Form, readOnly bool) error {
	for key, value := range form {
		// Check if the value is a string
		if str, ok := value.(string); ok {
			err := r.SetTextFieldByName(key, str, readOnly)
			if err != nil {
				return err
			}
		} else if boolean, ok := value.(bool); ok {
			va := "0"
			if boolean {
				va = "1"
			}
			err := r.SetCheckboxFieldByName(key, va, readOnly)
			if err != nil {
				return err
			}
		} else {
			fmt.Printf("Key: %s, Value: %v (unknown type)\n", key, value)
		}
	}
	return nil
}

func (r *PDFFormFiller) WriteToBytes() (data []byte, err error) {
	buffer := bytes.NewBuffer(nil)
	err = api.WriteContext(r.ctx, buffer)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (r *PDFFormFiller) extractFormFields() error {
	for objectId, item := range r.ctx.XRefTable.Table {
		dict, ok := item.Object.(types.Dict)
		if !ok {
			continue
		}
		subtype, ok := dict.Find("Subtype")
		if !ok || subtype.String() != "Widget" {
			continue
		}
		fieldType, ok := dict.Find("FT")
		if !ok {
			continue
		}
		var name string
		var err error
		dictT := dict["T"]
		if dictTHexLiteral, ok := dictT.(types.HexLiteral); ok {
			name, err = types.HexLiteralToString(dictTHexLiteral)
			if err != nil {
				return errors.Wrap(err, "Decode T attribute of form field failed")
			}
		} else if dictTStringLiteral, ok := dictT.(types.StringLiteral); ok {
			name = string(dictTStringLiteral)
		} else {
			panic("dict os not HexLiteral and not StringLiteral")
		}

		bb, err := r.ctx.RectForArray(dict.ArrayEntry("Rect"))
		if err != nil {
			return err
		}
		field := &FormField{
			ObjectId: objectId,
			Name:     name,
			Type:     fieldType.String(),
			Rect:     bb,
			Dict:     dict,
		}
		r.fieldMapById[objectId] = field
		r.fieldMapByName[name] = field
	}
	return nil
}

// FillFormFieldsWithItsIdName As a tool method, all text fields in the Form will be filled with their IDs to facilitate writing business logic.
func (r *PDFFormFiller) FillFormFieldsWithItsIdName() {
	for objectId, field := range r.fieldMapById {
		label := fmt.Sprintf("#%d %s", objectId, field.Name)
		r.AddText(1, label, int(field.Rect.LL.X), int(field.Rect.LL.Y))
	}
}

// GetFormDictById Get an element
func (r *PDFFormFiller) GetFormDictById(objectId int) (dict types.Dict, err error) {
	item, ok := r.ctx.XRefTable.Table[objectId]
	if !ok {
		return nil, errors.Errorf("field %d not found!", objectId)
	}
	dict, ok = item.Object.(types.Dict)
	if !ok {
		return nil, errors.Errorf("field %d, %s is not Dict type!", objectId, item.Object)
	}
	return dict, nil
}

// SetTextFieldByName
func (r *PDFFormFiller) SetTextFieldByName(name string, value string, setReadOnly bool) (err error) {
	formField, ok := r.fieldMapByName[name]
	if !ok {
		return errors.Wrapf(err, "Can not found Field: %s", name)
	}
	return r.SetTextFieldById(formField.ObjectId, value, setReadOnly)
}

// SetTextFieldById fills the text form
// objectId: Number the object (can be viewed through the mupdf tool `mutool show some.pdf form`)
// value: is the text content
func (r *PDFFormFiller) SetTextFieldById(objectId int, value string, setReadOnly bool) (err error) {
	formField, ok := r.fieldMapById[objectId]
	if !ok {
		return errors.Wrapf(err, "Can not found objectId %d", objectId)
	}
	if formField.Type != "Tx" {
		return errors.Errorf("type of field %d is %s (expected Tx)", objectId, formField.Type)
	}
	formField.Dict["V"] = types.NewHexLiteral([]byte(types.EncodeUTF16String(value)))

	if setReadOnly {
		formField.Dict["Ff"] = types.Integer(1)
	}
	return nil
}

// SetCheckboxFieldByName sets checkbox form options
func (r *PDFFormFiller) SetCheckboxFieldByName(name string, value string, setReadOnly bool) (err error) {
	formField, ok := r.fieldMapByName[name]
	if !ok {
		return errors.Wrapf(err, "Can not found Field: %s", name)
	}
	return r.SetCheckboxFieldById(formField.ObjectId, value, setReadOnly)
}

// SetCheckboxFieldById sets checkbox form options
// objectId: Number the object (can be viewed through the mupdf tool `mutool show some.pdf form`)
// value is the form status, and the optional options are the options defined by the AP attribute
func (r *PDFFormFiller) SetCheckboxFieldById(objectId int, value string, setReadOnly bool) (err error) {
	formField, ok := r.fieldMapById[objectId]
	if !ok {
		return errors.Wrapf(err, "Can not found objectId %d", objectId)
	}
	if formField.Type != "Btn" {
		return errors.Errorf("type of field %d is %s (expected Btn)", objectId, formField.Type)
	}

	// checkbox controls the display style through AS, and its options are defined in AP
	// https://www.verypdf.com/document/pdf-format-reference/index.htm
	formField.Dict["AS"] = types.Name(value)

	if setReadOnly {
		formField.Dict["Ff"] = types.Integer(1)
	}
	return nil
}

// AddImageOverObjectByName adds an image above an object
func (r *PDFFormFiller) AddImageOverObjectByName(name string, image []byte) (err error) {
	formField, ok := r.fieldMapByName[name]
	if !ok {
		return errors.Wrapf(err, "Can not found Field: %s", name)
	}
	return r.AddImageOverObjectById(formField.ObjectId, image)
}

func (r *PDFFormFiller) AddImageOverObjectById(objectId int, image []byte) (err error) {
	formField, ok := r.fieldMapById[objectId]
	if !ok {
		return errors.Wrapf(err, "Can not found objectId %d", objectId)
	}
	// FIXME: For PDFs with more than one page, need to calculate which page the objectId is on
	err = r.AddImage(1, image, int(formField.Rect.LL.X), int(formField.Rect.LL.Y), int(formField.Rect.Width()), int(formField.Rect.Height()), 1)
	if err != nil {
		return err
	}
	return nil
}

// AddImage adds an image to the specified area of the PDF
func (r *PDFFormFiller) AddImage(page int, image []byte, x, y, w, h int, scale float64) (err error) {
	pages := types.IntSet{
		page: true,
	}

	descriptionString := fmt.Sprintf("pos:bl, rot: 0, sc: %.4f abs, off: %d %d", scale, x, y)
	wm, err := api.ImageWatermarkForReader(bytes.NewReader(image), descriptionString, true, false, types.POINTS)
	if err != nil {
		return errors.Wrap(err, "Build ImageWatermark failed")
	}

	err = pdfcpu.AddWatermarks(r.ctx, pages, wm)
	if err != nil {
		return errors.Wrap(err, "Add ImageWatermark failed")
	}
	return err
}

// AddText adds text at the specified position in the PDF
func (r *PDFFormFiller) AddText(page int, text string, x, y int) (err error) {
	pages := types.IntSet{
		page: true,
	}
	descriptionString := fmt.Sprintf("points:12, strokec:#E00000, fillc:#E00000, sc: 1 abs, pos:bl, rot:0, off: %d %d", x, y)
	fmt.Printf("descriptionString %s\n", descriptionString)
	wm, err := api.TextWatermark(text, descriptionString, true, false, types.POINTS)
	if err != nil {
		return errors.Wrap(err, "Build TextWatermark failed")
	}
	err = pdfcpu.AddWatermarks(r.ctx, pages, wm)
	if err != nil {
		return errors.Wrap(err, "Add TextWatermark failed")
	}
	return err
}
