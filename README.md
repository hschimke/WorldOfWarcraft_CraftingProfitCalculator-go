# WorldOfWarcraft_CraftingProfitCalculator-go

WorldOfWarcraft_CraftingProfitCalculator is designed to calculate the potential profits for a given item, profession list, and realm. It does this by searching through all availble recipes and calculating the cast to produce them given current costs and materials. Some items are only craftable by other craftable items, so this process searches though each reagent needed by a recipe.

## Programs
### WorldOfWarcraft_CraftingProfitCalculator-go
CLI interface for the library. Has full functionality compared to the web interface.
### auction_archive_ctrl
Helper program to modify the behaviour of the auction archive and scanning tool.
### hourly_injest
Program to scan and evaluate auction houses for sales data.
### run_worker
Perform CPC runs for the React Web Client.
## Web Server
Simple server to handle the site and API.
## React Web Client
Web interface for CPC, relies on the Web Server for API and backend support.

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