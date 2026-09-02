package espncfb

import "github.com/jaslrobinson/golazo/internal/api"

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

// conferenceByID looks up a conference by its ESPN group ID (confirmed live
// via team.conferenceId). Returns a League with just the ID (empty name) for
// an ID not in fbsConferences — e.g. an FCS or Group of 5 conference not
// worth hardcoding — so LeagueTable can still be attempted.
func conferenceByID(id int) api.League {
	for _, c := range fbsConferences {
		if c.ID == id {
			return c
		}
	}
	return api.League{ID: id}
}
