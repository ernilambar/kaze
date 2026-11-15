// PaletteItem is one selectable row shared by Quick Open and the command
// palette. The id is what the palette hands back; the name and detail are what
// the user sees and what the ranking matches against.
export interface PaletteItem {
    id: string;
    name: string;
    detail: string;
}
