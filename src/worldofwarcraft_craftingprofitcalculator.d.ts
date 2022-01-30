// blizz_oath.ts
interface AccessToken {
    access_token: string,
    token_type: string,
    expires_in: number,
    scope: string,
    fetched: number,
    checkExpired: () => boolean
}

type ApiAuthorization = Readonly<{
    getAuthorizationToken: (region: RegionCode) => Promise<AccessToken>;
}>

// blizzard-api-call.ts
interface ApiConfig {
    connection_per_window?: number,
    window_size?: number
}

type CPCApi = Readonly<{
    getBlizzardAPIResponse: (region_code: RegionCode, data: string | Record<string, string | number>, uri: string) => Promise<BlizzardApi.BlizzardApiReponse | void>;
    getBlizzardRawUriResponse: (data: string | Record<string, string | number>, uri: string, region: RegionCode) => Promise<BlizzardApi.BlizzardApiReponse | void>;
    shutdownApiManager: () => void;
}>;

// blizzard-api-helpers.ts
interface AuctionPriceSummaryRecord {
    data?: SalesCountSummaryPrice[],
    min_value: number,
    max_value: number,
    avg_value: number
}

interface SalesCountSummary {
    sales_at_price: number,
    quantity_at_price: number
}

interface SalesCountSummaryPrice extends SalesCountSummary {
    price: number
}

interface AuctionSummaryData {
    min: number;
    max: number;
    avg: number;
    latest: number;
    price_map: Record<string | number, AuctionPriceSummaryRecord>;
    archives: {
        timestamp: string;
        data: SalesCountSummaryPrice[];
        min_value: number;
        max_value: number;
        avg_value: number;
    }[];
}

type SkillTierCyclicLinks = Record<number, {
    id: number,
    takes: number,
    makes: number
}[]>;

type SkillTierCyclicLinksBuild = {
    id: number[],
    quantity: number
}[][];

type CPCApiHelpers = Readonly<{
    getItemId: (region: RegionCode, item_name: ItemName) => Promise<ItemID>;
    getConnectedRealmId: (server_name: RealmName, server_region: RegionCode) => Promise<ConnectedRealmID>;
    getItemDetails: (item_id: ItemID, region: RegionCode) => Promise<BlizzardApi.Item>;
    getBlizProfessionsList: (region: RegionCode) => Promise<BlizzardApi.ProfessionsIndex>;
    getBlizProfessionDetail: (profession_id: number, region: RegionCode) => Promise<BlizzardApi.Profession>;
    getBlizConnectedRealmDetail: (connected_realm_id: ConnectedRealmID, region: RegionCode) => Promise<BlizzardApi.ConnectedRealm>;
    getBlizSkillTierDetail: (profession_id: number, skillTier_id: number, region: RegionCode) => Promise<BlizzardApi.ProfessionSkillTier>;
    getBlizRecipeDetail: (recipe_id: number, region: RegionCode) => Promise<BlizzardApi.Recipe>;
    checkIsCrafting: (item_id: ItemID, character_professions: Array<CharacterProfession>, region: RegionCode) => Promise<CraftingStatus>;
    buildCyclicRecipeList: (region: RegionCode) => Promise<SkillTierCyclicLinks>;
    getAuctionHouse: (server_id: ConnectedRealmID, server_region: RegionCode) => Promise<BlizzardApi.Auctions>;
    getCraftingRecipe: (recipe_id: number, region: RegionCode) => Promise<BlizzardApi.Recipe>;
}>;


// cached-data-sources.ts
type BonusesCache = Record<number | string, {
    id: number,
    level?: number,
    quality?: number,
    socket?: number
}>;

type StaticSources = {
    bonuses_cache: BonusesCache;
    rank_mappings_cache: RankMappingsCache;
    shopping_recipe_exclusion_list: ShoppingRecipeExclusionList;
    (): void;
}

interface RankMappingsCache {
    available_levels: number[],
    rank_mapping: number[]
}

interface ShoppingRecipeExclusionList {
    exclusions: number[]
}

interface StaticCacheConfig {
    cache_folder: string
}

type CPCCache = Readonly<{
    cacheCheck: (namespace: string, key: string | number) => Promise<boolean>;
    cacheGet: (namespace: string, key: string | number) => Promise<any>;
    cacheSet: (namespace: string, key: string | number, data: any, expiration_period: number) => Promise<void>;
    shutdown: () => Promise<void>;
}>

// auction-history.ts
type CPCAuctionHistory = Readonly<{
    scanRealms: () => Promise<void>;
    addRealmToScanList: (realm_name: RealmName, realm_region: RegionCode) => Promise<void>;
    removeRealmFromScanList: (realm_name: RealmName, realm_region: RegionCode) => Promise<void>;
    getAuctions: (item: ItemSoftIdentity, realm: ConnectedRealmSoftIentity, region: RegionCode, bonuses: number[] | string[] | string, start_dtm: number | string | undefined, end_dtm: number | string | undefined) => Promise<AuctionSummaryData>;
    getAllBonuses: (item: ItemSoftIdentity, region: RegionCode) => Promise<GetAllBonusesReturn>;
    archiveAuctions: () => Promise<void>;
    fillNItems: (fill_count?: number) => Promise<void>;
    fillNNames: (fillCount?: number) => Promise<void>;
    getAllNames: () => Promise<string[]>;
    getScanRealms: () => Promise<{ realm_names: string, realm_id: ConnectedRealmID, region: RegionCode }[]>;
}>;

interface GetAllBonusesReturn {
    bonuses: Record<string, string>[]
    item: BlizzardApi.Item
}

// databases.ts
interface DatabaseConfig {
    type: string,
    sqlite3?: {
        cache_fn: string,
        auction_fn: string
    }
}

type CPCDB = Readonly<{ getDb: (db_name: string) => Promise<DatabaseManagerFunction>, shutdown: () => Promise<void> }>;

type DatabaseClientFunction = {
    release: () => void;
    query: <Row>(query: string, values?: Array<string | number | boolean | null>) => Promise<{ rows: Row[] }>;
    (): void;
}
type DatabaseManagerFunction = {
    //db: any;
    //pool: any;
    db_type: string;
    get: <Row>(query: string, values?: Array<string | number | boolean | null>) => Promise<Row>;
    run: (query: string, values?: Array<string | number | boolean | null>) => Promise<void>;
    all: <Row>(query: string, values?: Array<string | number | boolean | null>) => Promise<Row[]>;
    query: <Row>(query: string, values?: Array<string | number | boolean | null>) => Promise<{ rows: Row[] }>;
    serialize: (query: Array<string>, values: Array<Array<string | number | boolean | null>>) => Promise<void>;
    getClient: () => Promise<DatabaseClientFunction>;
    (): void;
};

type Sqlite3DatabaseManagerFunction = {
    db: any;
} & DatabaseManagerFunction;

type PostgresDatabaseManagerFunction = {
    pool: any;
} & DatabaseManagerFunction;

// wow_crafting_profits.ts
interface AHItemPriceObject {
    total_sales: number,
    average: number,
    high: number,
    low: number,
}

interface CraftingStatus {
    recipe_ids: number[],
    craftable: boolean,
    recipes: {
        recipe_id: number,
        crafting_profession: CharacterProfession,
    }[]
}

interface ProfitAnalysisObject {
    item_id: number,
    item_name: string,
    ah_price: AHItemPriceObject,
    item_quantity: number,
    vendor_price: number,
    crafting_status: CraftingStatus
    bonus_lists: number[][],
    recipe_options: {
        prices: ProfitAnalysisObject[],
        recipe: {
            recipe_id: number
        },
        rank: number,
        rank_ah: AHItemPriceObject
    }[],
    bonus_prices: {
        level: number,
        ah: AHItemPriceObject
    }[]
}

interface ShoppingList {
    quantity: number,
    id: ItemID,
    name: ItemName,
    cost: {
        vendor: number,
        ah: OutputFormatPrice
    }
}

interface OutputFormatPrice {
    sales: number,
    high: number,
    low: number,
    average: number
}

type OutputFormatShoppingList = Record<number | string, ShoppingList[]>;

interface OutputFormatObject {
    name: string,
    id: number,
    required: number,
    recipes: {
        name: string,
        rank: number,
        id: number,
        output: {
            min: number,
            max: number,
            value: number
        },
        ah: OutputFormatPrice,
        high: number,
        low: number,
        average: number,
        parts: OutputFormatObject[]
    }[],
    ah: OutputFormatPrice,
    vendor: number,
    bonus_prices: {
        level: number,
        ah: OutputFormatPrice
    }[],
    shopping_lists: OutputFormatShoppingList
}

interface RunReturn {
    price?: ProfitAnalysisObject,
    intermediate?: OutputFormatObject,
    formatted: string
}

// RunConfiguration.js
interface AddonData {
    inventory: {
        id: ItemID,
        quantity: number
    }[],
    professions: CharacterProfession[],
    realm: {
        region_id?: number,
        region_name: string,
        realm_id?: ConnectedRealmID,
        realm_name: RealmName
    }
}

// server.js
interface SeenItemBonusesReturn {
    bonuses: Record<string, string>[],
    mapped: {
        text: string,
        parsed: Array<number | string>,
        reduced: string | undefined,
    }[],
    collected: {
        ilvl: {
            id: string | number,
            level: number,
        }[],
        socket: {
            id: string | number,
            sockets: number | undefined,
        }[],
        quality: {
            id: string | number,
            quality: number | undefined,
        }[],
        unknown: (string | number)[],
        empty: boolean,
    }
}

interface ServerErrorReturn {
    ERROR: string
}
type AuctionHistoryReturn = AuctionSummaryData;
type ServerRunResultReturn = OutputFormatObject;

// Global
type RegionCode = 'us' | 'eu' | 'kr' | 'tw';
type ItemID = number;
type ItemName = string;
type ItemSoftIdentity = ItemID | ItemName;

type ConnectedRealmID = number;
type RealmName = string;
type ConnectedRealmSoftIentity = ConnectedRealmID | RealmName;

type CharacterProfession = 'Jewelcrafting' | 'Tailoring' | 'Alchemy' | 'Herbalism' | 'Inscription' | 'Enchanting' | 'Blacksmithing' | 'Mining' | 'Engineering' | 'Leatherworking' | 'Skinning' | 'Cooking';