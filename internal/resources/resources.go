// Package resources holds the embedded files, go:embed only reaches its own
// directory.
package resources

import _ "embed"

// PickupPopupCode : shows a name popup on ground-item pickup.
//
//go:embed pickup_popup_code.bin
var PickupPopupCode []byte

// PickupPopupCodeWithIndex is PickupPopupCode plus " (#N)", N being the
// item's slot index (matches the spoiler log's numbering).
//
//go:embed pickup_popup_code_with_index.bin
var PickupPopupCodeWithIndex []byte

// DebugHUDBootstrap loads DebugHUDPayload from the disc, 5 sectors at most
// (kfconst.SlusDebugHUDPayloadMaxSize).
//
//go:embed debug_hud_bootstrap.bin
var DebugHUDBootstrap []byte

// DebugHUDPayload is the live debug menu and HUD, opened with L3+R3.
//
//go:embed debug_hud_payload.bin
var DebugHUDPayload []byte

// LevitateStub zeroes vertical velocity while the menu's Levitate row is on.
//
//go:embed levitate_stub.bin
var LevitateStub []byte

// MapNamesCSV maps each .map file to a human-readable display name.
// uses names from the official kf4 guide pdf mostly
//
//go:embed map_names.csv
var MapNamesCSV []byte

// TitleRandomizedOverlayPNG is pasted onto the title screen's logo2
// texture at randomize time.
//
//go:embed title_randomized_overlay.png
var TitleRandomizedOverlayPNG []byte

// SharedItemModel is the .omd every randomized pickup renders as.
// had to make a custom model since model loading is annoying.
//
//go:embed item_model.omd
var SharedItemModel []byte

// SharedItemTexture is the GS upload packet paired with SharedItemModel.
//
//go:embed item_texture.bin
var SharedItemTexture []byte
