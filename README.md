# WorldOfWarcraft_CraftingProfitCalculator-go

WorldOfWarcraft_CraftingProfitCalculator is designed to calculate the potential profits for a given item, profession list, and realm. It does this by searching through all availble recipes and calculating the cast to produce them given current costs and materials. Some items are only craftable by other craftable items, so this process searches though each reagent needed by a recipe.

## Programs

### WorldOfWarcraft_CraftingProfitCalculator-go
CLI interface for the library. Has full functionality compared to the web interface.

Options
 * `region`: The region to scan in. The default is US.
 * `server`: The server to scan. The default is "Hyjal".
 * `profession`: The list of professions to search, formatted as a JSON array of strings. The default is no professions. This setting is only used if `allprof` is set to false, so only use it if you know which professions you want to scan.
 * `item`: The name or ID of the item to search for. The default is "Grim-Veiled Bracers".
 * `count`: The number of items needed, this impacts costs estimates and builds.
 * `json_data`: A JSON object output by the wow addon in a string. Used for inventory control and profession overrides.
 * `json`: Use the data from `json_data` as the primary source, otherwise professions and realm/region are ignored.
 * `allprof`: Use all professions, including some that are specific to characters. The default is true.

### auction_archive_ctrl
Helper program to modify the behaviour of the auction archive and scanning tool.

Availble program modes are:
 * `add_scan_realm`: Add a realm to the scan realms list. The realm can be specified by either including `realm_name` or  `realm_id`, but never both. `region` is required and must be one of US, EU, KR, TW.
 * `archive_auctions`: Perform an archive auction. Current implementation deletes all data older than two weeks.
 * `fill_n_items`: Fill `count` items that have not been CPC scanned in the items table, items which have been scanned will have their names filled and their crafting status set for a default case.
 * `fill_n_names`: Fill `count` items which do no thave names in the items table with names. This is used by the React Web Client to autofill item boxes.
 * `get_all_bonuses`: Return all seen bonuses for an item identified by either `item_name` or `item_id` within a given `region`. This is used by the React Web Client to fill the auction search boxes.
 * `get_all_names`: Get a deduplicated list of all names in the items table.
 * `get_auctions`: Perform an auction history search given: `realm_name`, `realm_id`, `region`, `item_name`, `item_id`, `end_dtm`, `bonuses`. All are optional, though searching without specifying any of them my have strange results.
 * `get_scan_realms`: Return a list of all realms in the scan realms table.
 * `remove_scan_realm`: Remove a realm to the scan realms list. The realm can be specified by either including `realm_name` or  `realm_id`, but never both. `region` is required and must be one of US, EU, KR, TW.
 * `scan_realms`: Perform a scan on all realms in the scan realms list. Realms can be added or removed with `add_scan_realm` and `remove_scan_realm`. This operation runs both item scan and auction scan.

 Availble data paramaters are:
 * `realm_name`: A string name for a realm. Do not include both `realm_name` and `realm_id`.
 * `realm_id`: The ID number for a connected realm. Do not include both `realm_name` and `realm_id`.
 * `region`: The region in which to check. US, EU, KR, TW are all supported.
 * `count`: A number indicating how high to count. This is used for `fill_n_names` and `fill_n_items`.
 * `item_name`: A string name for an item. Do not include both `item_name` and `item_id`.
 * `item_id`: The ID number for a blizzard item. Do not include both `item_name` and `item_id`.
 * `start_dtm`: A date string. Used only for auction searches.
 * `end_dtm`: A date string. Used only for auction searches.
 * `bonuses`: A JSON string showing bonuses to search for.  Used only for auction searches.
 * `log_level`: Override the default log level set in the environment.

### hourly_injest
Program to scan and evaluate auction houses for sales data. hourly_injest can be run in several modes. If running via a cron job or SystemD schedule the environment variable `STANDALONE_CONTAINER` must be set to "hourly". When running as a daemon or in a docker container, `STANDALONE_CONTAINER` must be set to "worker".

The program performs several functions:
 * Once every three hours id downloads all registered scan realms and stores their auctions for historical analysis.
 * Every few minutes it downloads a list of items and fills their names in the database. This is used by several functions in the React Web Client.
 * Every few minutes it checks to see if a set of items is craftable, building up a cache of those results.
 * Once a day it deletes all auction data older than three weeks.

If scheduling the job with cron or SystemD it is important to have it run once per hour. If running in another mode it will handle the scheduling itself.

### run_worker
Perform CPC runs for the React Web Client. run_worker monitors the job queue from the website and performs CPC runs on website input. When the job is complete it sends it back into the queue to be picked up by the website.

## Web Server
Simple server to handle the site and API. The server is best used within a docker container, though it can be run anywhere. It requires the standard set of CPC files and folders, and is intended to serve the React Web Client as well as the API.

## React Web Client
Web interface for CPC, relies on the Web Server for API and backend support.

## World of Warcraft AddOn

### Installing the World of Warcraft AddOn
To use the inventory for a character (or set of characters) when computing the shopping list, the option AddOn must be installed and used to generate json data. The AddOn can be found in the `wow-addon` folder in the root of the repository. Copy the folder `CraftingProfitCalculator_data` in that directory to the `AddOns` folder in your World of Warcraft installation. The AddOn can be downloaded from a website built using the docker build instructions.

#### Using the AddOn
The AddOn provides three slash commands within World of Warcraft.
* `/cpcr`: To run the inventory scan in the background. This should be done for each character.
* `/cpcc`: Runs an inventory scan and outputs the json data for the currently logged in character.
* `/cpca`: Runs an inventory scan and outputs the json data for all scanned characters.

Once the json data is collected, it can be coppied into the web page provided by the server or into an option in the CLI program. JSON data is only refreshed when one of the above commands is written, so if a character has changed the contents of their inventory since the last run it will not be reflected.

The AddOn also collection region, realm, and profession data for all scanned characters. This can be used for running the program without specifying all of the parameters directly, instead infering them from the provided AddOn output.

#### Using the AddOn with Reagent Bank and Main Bank
In order to include the contents of a character's bank and reagent bank the above slash commands must be run while the character's bank is open.

## Environment Variables
There are several required environment variables for the assorted programs and systems within CPC. 
 * `CLIENT_ID` Client ID for the blizzard API application
 * `CLIENT_SECRET` Client Secret for the blizzard API applicatoin
 * `LOG_LEVEL` Logging level, default is "info"
 * `SERVER_PORT` Port on which the server should attach
 * `REDIS_URL` The connection string for redis
 * `STANDALONE_CONTAINER` Standalone container can be "hourly" "worker" "standalone" or "normal". This should always be set to "worker" for the hourly_injest program when run in docker or as a daemon, and always to "hourly" if run with a scheduler, such as cron or SystemD.
 * `DISABLE_AUCTION_HISTORY` Set to true to disable auction history, default is false
 * `DATABASE_CONNECTION_STRING` Connection string to the postgres database.

 ## Requirements
 CPC requires Redis and Postgres, as well as a Client ID and Client Secret from Blizzard.

 ## Building
 The program can be built by entering each of the directories in the `cmd` folder and running `go build`. The primary reason one would do this would be for running the CLI.

 Alternatively, the entire system can be run within docker, this is the prefered way to run the program for most situations. The docker containers can be built by running `sh scripts/docker-build.sh` from within the root of the repository. An example docker compose file is available in `docker/cpc/docker-compose.yaml`.