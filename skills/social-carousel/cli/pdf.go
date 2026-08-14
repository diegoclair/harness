package main

import (
	"fmt"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// combinePDF assembles the provided PNG files into a single multi-page PDF,
// one PNG per page, sized to the platform's LOGICAL dimensions (1080×1350 pt
// for 4:5) while keeping the full-resolution 2x image data embedded.
//
// Two steps are required because pdfcpu v0.12.1's ImportImagesFile sizes each
// page to the source image's pixel dimensions and ignores the import PageDim —
// so a 2160×2700px retina PNG yields a 2160×2700pt page (~76×95cm). LinkedIn
// re-frames such oversized pages incorrectly (stacking two slides per card).
// We import first, then Resize the pages to the logical PageDim: the image is
// re-scaled by reference (pixels preserved, still crisp), only the MediaBox
// shrinks to the size the platform expects.
func combinePDF(pngPaths []string, outPath string, spec PlatformSpec) error {
	if len(pngPaths) == 0 {
		return fmt.Errorf("combinePDF: no PNG files provided")
	}

	imp := buildImportConfig(spec)

	// Step 1: import into a temp PDF (pages come out at image pixel size).
	tmp, err := os.CreateTemp("", "carousel-*.pdf")
	if err != nil {
		return fmt.Errorf("combinePDF: temp file: %w", err)
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpPath)

	// ImportImagesFile appends when outFile exists; an empty file fails to open.
	// Remove the placeholder so pdfcpu creates a fresh PDF.
	if err := os.Remove(tmpPath); err != nil {
		return fmt.Errorf("combinePDF: clear temp: %w", err)
	}

	if err := api.ImportImagesFile(pngPaths, tmpPath, imp, nil); err != nil {
		return fmt.Errorf("combinePDF: pdfcpu import: %w", err)
	}

	// Step 2: resize every page to the logical dimensions the platform expects.
	res := &model.Resize{
		PageDim: &types.Dim{Width: float64(spec.Width), Height: float64(spec.Height)},
		UserDim: true,
		Unit:    types.POINTS,
	}
	if err := api.ResizeFile(tmpPath, outPath, nil, res, nil); err != nil {
		return fmt.Errorf("combinePDF: pdfcpu resize: %w", err)
	}
	return nil
}

// buildImportConfig constructs a pdfcpu Import configuration that sizes
// each page to match the carousel's logical dimensions.
//
// PDF points at 72 DPI: 1 logical px = 1 pt (both are at 72 DPI).
// The image is scaled to fill the entire page (Pos=Full, Scale=1.0,
// ScaleAbs=false) so no white margins appear.
func buildImportConfig(spec PlatformSpec) *pdfcpu.Import {
	// 1 logical CSS px = 1 PDF point (at 72 DPI).
	wPts := float64(spec.Width)
	hPts := float64(spec.Height)

	imp := pdfcpu.DefaultImportConfig()

	// Override the page size with the exact platform dimensions.
	imp.PageDim = &types.Dim{
		Width:  wPts,
		Height: hPts,
	}
	imp.UserDim = true
	imp.PageSize = ""

	// Scale=1.0 + ScaleAbs=false means "scale relative to page, fill it".
	// Pos=Full means the image fills the entire page with no anchor offset.
	imp.Scale = 1.0
	imp.ScaleAbs = false
	imp.Pos = types.Full

	return imp
}
