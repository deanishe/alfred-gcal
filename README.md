
<div align="center">
    <img height="128" width="128" src="https://raw.githubusercontent.com/deanishe/alfred-gcal/master/icons/icon.png">
</div>

Google Calendar for Alfred
==========================

View Google Calendar events in [Alfred][alfred]. Supports multiple accounts.

<!-- MarkdownTOC autolink="true" bracket="round" depth="3" autoanchor="true" -->

- [Google Calendar for Alfred](#google-calendar-for-alfred)
  - [Download & installation](#download--installation)
  - [Usage](#usage)
    - [Date format](#date-format)
    - [Add event format](#add-event-format)
  - [Configuration](#configuration)
  - [Licensing & thanks](#licensing--thanks)
  - [Privacy](#privacy)

<!-- /MarkdownTOC -->


<a name="download--installation"></a>
Download & installation
-----------------------

Grab the workflow from [GitHub releases][download]. Download the `Google-Calendar-View-X.X.alfredworkflow` file and double-click it to install.


<a name="usage"></a>
Usage
-----

When run, the workflow will open Google Calendar in your browser and ask for permission to access your calendars. If you do not grant permission, it won't work. The workflow requests permission to edit your calendars, as this is needed for the "Add New Event" feature (keyword `gnew`). It does not otherwise alter your calendars or events in any way.

You will also be prompted to activate some calendars (the workflow will show events from these calendars). You can alter the active calendars or add/remove Google accounts in the settings using keyword `gcalconf`.

- `gcal` — Show upcoming events.
    - `<query>` — Filter list of events.
    - `↩` — Open event in browser or day in workflow.
    - `⌘↩` — Open event in Google Maps or Apple Maps (if event has a location).
    - `⇧` / `⌘Y` — Quicklook event details.
- `today` / `tomorrow` / `yesterday` — Show events for the given day.
    - `<query>` / `↩` / `⌘↩` / `⇧` / `⌘Y` — As above.
- `gdate [<date>]` — Show one or more dates. See below for query format.
    - `↩` — Show events for the given day.
- `gnew [<query>]` — Add a new event in the one of active calendars. (example: Some meeting at Office at 5pm with Ian)
    - `↩` — Create event in selected calendar.
- `gcalconf [<query>]` — Show workflow configuration.
    - `Active Calendars…` — Turn calendars on/off.
        - `↩` — Toggle calendar on/off.
    - `Add Account…` — Add a Google account.
        - `↩` — Open Google login in browser to authorise an account.
    - `your.email@gmail.com` — Your logged in Google account(s).
        - `↩` — Remove account.
    - `Open Locations in Google Maps/Apple Maps` — Choose app to open event locations.
        - `↩` — Toggle setting between Google Maps & Apple Maps.
    - `Workflow is up to Date` / `An Update is Available` — Whether a newer version of the workflow is available.
        - `↩` — Check for or install update.
    - `Open Locations in XYZ` — Open locations in Google Maps or Apple Maps.
    - `↩` — Toggle between applications.
    - `Open Documentation` — Open this page in your brower.
    - `Get Help` — Visit [the thread for this workflow][forumthread] on [AlfredForum.com][alfredforum].
    - `Report Issue` — [Open an issue][issues] on GitHub.
    - `Clear Cached Calendars & Events` — Remove cached lists of calendars and events.


<a name="date-format"></a>
### Date format ###

When viewing dates/events, you can specify and jump to a particular date using the following input format:

- `YYYY-MM-DD` — e.g. `2017-12-01`
- `YYYYMMDD` — e.g. `20180101`
- `[+|-]N[d|w]` — e.g.:
    - `1`, `1d` or `+1d` for tomorrow
    - `-1` or `-1d` for yesterday
    - `3w` for 21 days from now
    - `-4w` for 4 weeks ago


<a name="add-event-format"></a>
### Add event format ###

The "Add New Event" feature (keyword `gnew`) creates an event using Google Calendar's natural language syntax. This doesn't appear to be properly documented anywhere, but it is pretty powerful. You can specify event title, location, time & duration and repetition. Some examples:

- `Wash pants` — creates an event titled "Wash pants" starting now using your default event duration
- `Clean pants party tomorrow` — creates an all-day event for tomorrow title "Clean pants party"
- `Drink beer every day 2000-2200` — creates an event titled "Drink beer" starting at 8pm, finishing at 10pm, and repeating every day.


<a name="configuration"></a>
Configuration
-------------

There are a couple of options in the workflow's configuration sheet (the `[x]` button in Alfred Preferences):

| Setting | Description |
|---------|-------------|
| `CALENDAR_APP` | Name of application to open Google Calendar URLs (not map URLs) in. If blank, your default browser is used. |
| `EVENT_CACHE_MINS` | Number of minutes to cache event lists before updating from the server. |
| `SCHEDULE_DAYS` | The number of days' events to show with the `gcal` keyword. |
| `APPLE_MAPS` | Set to `1` to open map links in Apple Maps instead of Google Maps. This option can be toggled from within the workflow's configuration with keyword `gcalconf`. |


<a name="licensing--thanks"></a>
Licensing & thanks
------------------

This workflow is released under the [MIT Licence][mit].

It is heavily based on the [Google API libraries for Go][google-libs] ([BSD 3-clause licence][google-licence]) and [AwGo][awgo] libraries ([MIT][mit]), and of course, [Google Calendar][gcal].


The icons are from or based on [Font Awesome][awesome] and [Weather Icons][weather] (both [SIL][sil]).

Special thanks to [@diffmike][diffmike] for adding the "Add New Event" feature.


<a name="privacy"></a>
Privacy
-------

The data used and accessed by this workflow are stored exclusively on your own Mac. Nothing is shared with anyone. When you authorise this workflow to access your Google Calendars, the only person you are enabling to read that data is you.

[gcal]: https://calendar.google.com/calendar/
[google-libs]: https://github.com/google/google-api-go-client
[google-licence]: https://github.com/google/google-api-go-client/blob/master/LICENSE
[alfred]: https://alfredapp.com/
[alfredforum]: https://www.alfredforum.com/
[awgo]: https://github.com/deanishe/awgo
[forumthread]: https://www.alfredforum.com/topic/11016-google-calendar-view/
[download]: https://github.com/deanishe/alfred-gcal/releases/latest
[issues]: https://github.com/deanishe/alfred-gcal/issues
[sil]: http://scripts.sil.org/cms/scripts/page.php?site_id=nrsi&id=OFL
[mit]: https://opensource.org/licenses/MIT
[awesome]: http://fortawesome.github.io/Font-Awesome/
[weather]: https://erikflowers.github.io/weather-icons/
[diffmike]: https://github.com/diffmike
