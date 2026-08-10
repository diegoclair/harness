package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/diegoclair/skills/pkg/atlassian/adf"
)

// resolveSpaceID turns --space-id into the numeric ID the v2 API requires.
// Empty falls back to the active space from config. A non-numeric value is
// treated as a space key and resolved via config or the API — it is never
// sent to Confluence as-is, which the API rejects (and which used to panic
// here on the nil result).
func resolveSpaceID(spaceID string, client *adf.ConfluenceClient) (string, error) {
	cfg := adf.ReadActiveConfig()

	if spaceID == "" {
		if cfg.SpaceID == "" {
			return "", fmt.Errorf("no --space-id given and no active space configured — run `confluence-docs space use <key>` or pass --space-id")
		}
		return cfg.SpaceID, nil
	}

	if _, numErr := strconv.Atoi(spaceID); numErr == nil {
		return spaceID, nil
	}

	if cfg.SpaceID != "" && strings.EqualFold(spaceID, cfg.SpaceKey) {
		return cfg.SpaceID, nil
	}

	spaces, _, err := getSpaces(client, false)
	if err != nil {
		return "", fmt.Errorf("--space-id %q is not numeric and could not be resolved as a space key: %w", spaceID, err)
	}
	for _, s := range spaces {
		if strings.EqualFold(s.Key, spaceID) {
			return s.ID, nil
		}
	}
	return "", fmt.Errorf("--space-id %q is not a numeric ID and does not match any accessible space key — run `confluence-docs space list` to see valid keys", spaceID)
}

func runPageCreate(args []string, stdout, stderr io.Writer) (int, error) {
	var spaceID, parentID, title, markdownFile, adfFile string
	var fullWidth, fixedWidth bool

	remaining, cloud, email, token, err := parseCommonPageFlags(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitInputErr, errInvalidUsage
	}

	for i := 0; i < len(remaining); i++ {
		a := remaining[i]
		switch a {
		case "--space-id":
			if i+1 >= len(remaining) {
				fmt.Fprintln(stderr, "--space-id requires a value")
				return exitInputErr, errInvalidUsage
			}
			spaceID = remaining[i+1]
			i++
		case "--parent-id":
			if i+1 >= len(remaining) {
				fmt.Fprintln(stderr, "--parent-id requires a value")
				return exitInputErr, errInvalidUsage
			}
			parentID = remaining[i+1]
			i++
		case "--title":
			if i+1 >= len(remaining) {
				fmt.Fprintln(stderr, "--title requires a value")
				return exitInputErr, errInvalidUsage
			}
			title = remaining[i+1]
			i++
		case "--markdown":
			if i+1 >= len(remaining) {
				fmt.Fprintln(stderr, "--markdown requires a file path")
				return exitInputErr, errInvalidUsage
			}
			markdownFile = remaining[i+1]
			i++
		case "--adf":
			if i+1 >= len(remaining) {
				fmt.Fprintln(stderr, "--adf requires a file path")
				return exitInputErr, errInvalidUsage
			}
			adfFile = remaining[i+1]
			i++
		case "--full-width":
			fullWidth = true
		case "--fixed-width":
			fixedWidth = true
		default:
			fmt.Fprintln(stderr, "unknown flag:", a)
			return exitInputErr, errInvalidUsage
		}
	}

	if parentID == "" {
		fmt.Fprintln(stderr, "page create: --parent-id is required")
		return exitInputErr, errInvalidUsage
	}
	if title == "" {
		fmt.Fprintln(stderr, "page create: --title is required")
		return exitInputErr, errInvalidUsage
	}
	if markdownFile != "" && adfFile != "" {
		fmt.Fprintln(stderr, "page create: specify either --markdown or --adf, not both")
		return exitInputErr, errInvalidUsage
	}
	if fullWidth && fixedWidth {
		fmt.Fprintln(stderr, "page create: --full-width and --fixed-width are mutually exclusive")
		return exitInputErr, errInvalidUsage
	}

	client, ok := buildClient(cloud, email, token, stderr)
	if !ok {
		return exitUnknownErr, nil
	}

	resolvedSpaceID, spaceErr := resolveSpaceID(spaceID, client)
	if spaceErr != nil {
		fmt.Fprintln(stderr, "page create:", spaceErr)
		return exitInputErr, spaceErr
	}

	var result *adf.PageCreateResult
	var createErr error

	if markdownFile != "" {
		src, rdErr := os.ReadFile(markdownFile)
		if rdErr != nil {
			fmt.Fprintln(stderr, "reading markdown:", rdErr)
			return exitInputErr, rdErr
		}
		if adf.RequiresStorageFormat(string(src)) {
			// Markdown contains :::properties or other storage-only macros.
			// Convert to Confluence storage XML and upload with representation=storage.
			// Use client-aware conversion so @handle mentions in :::properties
			// are resolved to real Confluence user mention links.
			storageBody, sErr := adf.MarkdownToStorageWithClient(src, client)
			if sErr != nil {
				fmt.Fprintln(stderr, "convert markdown to storage:", sErr)
				return exitParseErr, sErr
			}
			result, createErr = client.CreatePageStorage(resolvedSpaceID, parentID, title, storageBody)
		} else {
			doc, cErr := adf.Convert(src)
			if cErr != nil {
				fmt.Fprintln(stderr, "parse markdown:", cErr)
				return exitParseErr, cErr
			}
			result, createErr = client.CreatePage(resolvedSpaceID, parentID, title, &doc)
		}
	} else if adfFile != "" {
		adfBytes, rdErr := os.ReadFile(adfFile)
		if rdErr != nil {
			fmt.Fprintln(stderr, "reading ADF:", rdErr)
			return exitInputErr, rdErr
		}
		doc, uErr := adf.UnmarshalDoc(adfBytes)
		if uErr != nil {
			fmt.Fprintln(stderr, "invalid ADF:", uErr)
			return exitParseErr, uErr
		}
		result, createErr = client.CreatePage(resolvedSpaceID, parentID, title, &doc)
	} else {
		result, createErr = client.CreatePage(resolvedSpaceID, parentID, title, nil)
	}

	if createErr != nil {
		fmt.Fprintln(stderr, "error:", createErr)
		return exitUnknownErr, createErr
	}
	if result == nil {
		err := fmt.Errorf("page create: API returned no error but no page result")
		fmt.Fprintln(stderr, err)
		return exitUnknownErr, err
	}

	// Apply page appearance (full-width / fixed-width) if requested.
	if fullWidth || fixedWidth {
		appearance := adf.PageAppearanceFullWidth
		if fixedWidth {
			appearance = adf.PageAppearanceFixedWidth
		}
		if appErr := client.SetPageAppearance(result.ID, appearance); appErr != nil {
			fmt.Fprintf(stderr, "warning: page created but appearance could not be set: %v\n", appErr)
		}
	}

	out := map[string]string{
		"pageId": result.ID,
		"title":  result.Title,
		"url":    client.PageURL(result.Links.WebUI),
	}
	outBytes, _ := json.MarshalIndent(out, "", "  ")
	fmt.Fprintln(stdout, string(outBytes))
	return exitOK, nil
}
