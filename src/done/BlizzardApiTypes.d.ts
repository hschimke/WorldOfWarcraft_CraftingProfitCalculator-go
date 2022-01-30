declare namespace BlizzardApi {
    interface ItemSearch {
        pageCount: number,
        page: number,
        results: {
            data: {
                name: Record<string, string>,
                id: ItemID
            }
        }[]
    }

    interface ConnectedRealmIndex {
        connected_realms: {
            href: string
        }[]
    }

    interface ConnectedRealm {
        id: ConnectedRealmID,
        realms: {
            name: string
        }[]
    }

    interface Item {
        id: ItemID,
        name: ItemName,
        description: string,
        purchase_price: number,
        purchase_quantity: number,
        level: number
    }

    interface ProfessionsIndex {
        professions: {
            name: string,
            id: number
        }[]
    }

    interface Profession {
        skill_tiers: {
            name: string,
            id: number,

        }[],
        name: string,
        id: number
    }

    interface ProfessionSkillTier {
        categories: {
            recipes: {
                id: number,
                name: string
            }[],
            name: string
        }[]
    }

    interface Recipe {
        id: number,
        name: string,
        alliance_crafted_item: {
            id: number
        },
        horde_crafted_item: {
            id: number
        },
        crafted_item: {
            id: number
        },
        reagents: Array<{
            reagent: {
                id: number
            },
            quantity: number
        }>,
        crafted_quantity: {
            minimum?: number,
            maximum?: number,
            value?: number
        }
    }

    interface Auctions {
        auctions: {
            item: {
                id: ItemID,
                bonus_lists: number[],
            }
            quantity: number,
            buyout: number,
            unit_price: number
        }[]
    }

    type BlizzardApiReponse = ItemSearch | ConnectedRealmIndex | Item | ProfessionsIndex | Profession | ConnectedRealm | ProfessionSkillTier | Recipe | Auctions;
}