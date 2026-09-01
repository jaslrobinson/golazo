package espncfb

import "github.com/0xjuanma/golazo/internal/api"

// fbsConferences is a hardcoded table of major FBS conference (ESPN "group")
// IDs, mirroring the static approach golazo already takes for FotMob league
// IDs (see internal/api/LEAGUE_IDS.md). ESPN's site API has no clean
// "list all conferences" endpoint, so these are captured from public
// reference rather than a live request — VERIFY each ID against
// /scoreboard?groups={id} before trusting it in the UI.
var fbsConferences = []api.League{
	{ID: 8, Name: "SEC", Country: "NCAA"},
	{ID: 5, Name: "Big Ten", Country: "NCAA"},
	{ID: 1, Name: "ACC", Country: "NCAA"},
	{ID: 4, Name: "Big 12", Country: "NCAA"},
	{ID: 151, Name: "American Athletic", Country: "NCAA"},
	{ID: 12, Name: "Conference USA", Country: "NCAA"},
	{ID: 15, Name: "MAC", Country: "NCAA"},
	{ID: 17, Name: "Mountain West", Country: "NCAA"},
	{ID: 37, Name: "Sun Belt", Country: "NCAA"},
	{ID: 18, Name: "FBS Independents", Country: "NCAA"},
}
