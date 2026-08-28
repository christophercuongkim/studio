package cli

// commands is the ordered command surface (plan §5). The order here is the
// order shown in `studio --help` — it follows the pipeline, not the alphabet.
//
// A command with a nil Run is declared but not yet built; running it exits 2
// with "not implemented yet". As each milestone lands, wire its constructor
// into the corresponding entry (e.g. Run: ingest.Command().Run) so the surface
// grows without touching the dispatcher.
var commands = []*Command{
	{Name: "new", Usage: "<slug>", Summary: "Create a new project folder and skeletons", Run: runNew},
	{Name: "ingest", Usage: "<dump-dir> --project <dir>", Summary: "Import a card dump into the project layout", Run: runIngest},
	{Name: "serve", Usage: "<project>", Summary: "Review UI: rate, name, keep/reject clips", Run: runServe},
	{Name: "apply", Usage: "<project>", Summary: "Rename kept clips to their final names", Run: runApply},
	{Name: "undo", Usage: "<project>", Summary: "Reverse the most recent apply", Run: runUndo},
	{Name: "scaffold", Usage: "<project>", Summary: "Generate a Kdenlive project from a template", Run: runScaffold},
	{Name: "status", Usage: "<project>", Summary: "Show review/apply counts for a project"},
	{Name: "search", Usage: "<text>", Summary: "Search clips across every ingested shoot"},
	{Name: "prompt", Usage: "<project>", Summary: "Bullet-script recording prompter"},
	{Name: "chapters", Usage: "<project>", Summary: "Kdenlive guides → YouTube chapters"},
	{Name: "qc", Usage: "<project>", Summary: "Pre-upload render checks"},
	{Name: "thumbs", Usage: "<project>", Summary: "Extract and rank thumbnail candidates"},
	{Name: "upload", Usage: "<project>", Summary: "Upload the render to YouTube"},
	{Name: "archive", Usage: "<project>", Summary: "Verify, cold-store, and prune a finished project"},
}
