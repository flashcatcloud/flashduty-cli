package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"
)

// Curated multipart-upload commands. Both upload endpoints consume
// multipart/form-data, which the generated --data/--flags body assembly cannot
// express, so the generators (SDK + CLI) skip these ops and the commands live
// here, attached to their generated path groups after registerGenerated.

func attachEnrichmentMappingDataUpload(root *cobra.Command) {
	g := genGroup(root, "enrichment", "On-call/Alert enrichment API")
	genAddLeaf(g, newMappingDataUploadCmd())
}

func attachSafariSkillUpload(root *cobra.Command) {
	g := genGroup(root, "safari", "AI SRE API")
	genAddLeaf(g, newSkillUploadCmd())
}

// openUploadFile resolves --file: a path on disk, or "-" for stdin (a single
// pipe read, consistent with --data -). The returned name feeds the multipart
// filename; stdin gets fallbackName. Callers close the ReadCloser (os.Stdin is
// safe to close here: the process exits right after).
func openUploadFile(path, fallbackName string) (io.ReadCloser, string, error) {
	if path == "-" {
		r, err := claimStdin("--file")
		if err != nil {
			return nil, "", err
		}
		return io.NopCloser(r), fallbackName, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, "", fmt.Errorf("cannot open --file: %w", err)
	}
	return f, filepath.Base(path), nil
}

func newMappingDataUploadCmd() *cobra.Command {
	var schemaID, filePath string
	var doNotTruncateFirst bool

	cmd := &cobra.Command{
		Use:   "mapping-data-upload --schema-id <id> --file <csv>",
		Short: "Upload mapping data via CSV",
		Long: `Upload a CSV file to bulk-load mapping data into a mapping schema.

By default the existing data is truncated before the new rows load; pass
--do-not-truncate-first to append instead. The CSV header row must include all
of the schema's source/result label names; the file may be at most 100 MB.
Use --file - to stream the CSV from stdin.

API: POST /enrichment/mapping/data/upload (mapping-data-write-upload)`,
		Example: `  flashduty enrichment mapping-data-upload --schema-id 665f1a2b3c4d5e6f7a8b9c01 --file rows.csv
  cat rows.csv | flashduty enrichment mapping-data-upload --schema-id 665f1a2b3c4d5e6f7a8b9c01 --file -`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				file, name, err := openUploadFile(filePath, "mapping.csv")
				if err != nil {
					return err
				}
				defer func() { _ = file.Close() }()

				_, err = ctx.Client.AlertEnrichment.MappingDataWriteUpload(cmdContext(ctx.Cmd), &flashduty.MappingDataUploadRequest{
					SchemaID:           schemaID,
					DoNotTruncateFirst: doNotTruncateFirst,
					File:               file,
					Filename:           name,
				})
				if err != nil {
					return err
				}
				ctx.WriteResult("OK: POST /enrichment/mapping/data/upload")
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&schemaID, "schema-id", "", "ID of the target mapping schema (ObjectID hex) (required)")
	cmd.Flags().StringVar(&filePath, "file", "", "CSV file path, or - for stdin (required)")
	cmd.Flags().BoolVar(&doNotTruncateFirst, "do-not-truncate-first", false, "Append to the existing rows instead of truncating them first")
	_ = cmd.MarkFlagRequired("schema-id")
	_ = cmd.MarkFlagRequired("file")
	return cmd
}

func newSkillUploadCmd() *cobra.Command {
	var filePath, skillID string
	var teamID int64
	var replace bool

	cmd := &cobra.Command{
		Use:   "skill-upload --file <archive>",
		Short: "Upload a skill archive",
		Long: `Upload a skill archive (.skill / .zip / .tar.gz / .tgz) to create or replace a skill.

By default the upload fails when a skill with the same name already exists;
pass --replace to overwrite it instead — matched by --skill-id when given,
otherwise by skill name. --team-id scopes the created skill to a team
(0 = account-wide). Use --file - to stream the archive from stdin.

API: POST /safari/skill/upload (skill-write-upload)`,
		Example: `  flashduty safari skill-upload --file my-skill.zip
  flashduty safari skill-upload --file my-skill.zip --replace --team-id 42`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				if skillID != "" && !replace {
					return fmt.Errorf("--skill-id only takes effect with --replace")
				}
				file, name, err := openUploadFile(filePath, "skill.zip")
				if err != nil {
					return err
				}
				defer func() { _ = file.Close() }()

				item, _, err := ctx.Client.Skills.WriteUpload(cmdContext(ctx.Cmd), &flashduty.SkillUploadRequest{
					File:     file,
					Filename: name,
					TeamID:   teamID,
					Replace:  replace,
					SkillID:  skillID,
				})
				if err != nil {
					return err
				}
				if ctx.Structured() {
					return ctx.Printer.Print(item, nil)
				}
				ctx.WriteResult(fmt.Sprintf("OK: skill %q uploaded (%s)", item.SkillName, item.SkillID))
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&filePath, "file", "", "Skill archive path (.skill/.zip/.tar.gz/.tgz), or - for stdin (required)")
	cmd.Flags().Int64Var(&teamID, "team-id", 0, "Team scope for the created skill (0 = account-wide)")
	cmd.Flags().BoolVar(&replace, "replace", false, "Overwrite the existing skill instead of failing on a name collision")
	cmd.Flags().StringVar(&skillID, "skill-id", "", "Existing skill ID to replace (requires --replace)")
	_ = cmd.MarkFlagRequired("file")
	return cmd
}
