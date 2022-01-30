import { parentLogger } from './logging.js';
const logger = parentLogger.child({});

class RunConfiguration {
    #internal_inventory: Record<ItemID, number> = {};
    #inventory_overlay: Record<ItemID, number> = {};
    #professions: CharacterProfession[] = [];
    #realm_name: RealmName = '';
    #realm_region: string = '';
    #item_id: ItemSoftIdentity
    #item_count: number

    constructor(raw_configuration_data: AddonData, item: ItemSoftIdentity, count: number) {
        if (raw_configuration_data != undefined) {
            for (let item of raw_configuration_data.inventory) {
                this.#internal_inventory[item.id] = Number(item.quantity);
            }
            for (let prof of raw_configuration_data.professions) {
                this.#professions.push(prof);
            }
            this.#realm_name = raw_configuration_data.realm.realm_name;
            this.#realm_region = raw_configuration_data.realm.region_name;
        }
        this.#item_id = item;
        this.#item_count = count;
    }
    get realm_name(): RealmName {
        return this.#realm_name;
    }
    get realm_region(): string {
        return this.#realm_region;
    }
    get professions(): Array<CharacterProfession> {
        return this.#professions;
    }
    get item_id(): ItemSoftIdentity {
        return this.#item_id;
    }
    get item_count(): number {
        return this.#item_count;
    }
    itemInInventory(item_id: ItemID): boolean {
        return item_id in this.#internal_inventory;
    }
    itemCount(item_id: ItemID): number {
        const is_in_inventory = this.itemInInventory(item_id);
        const has_overlay = item_id in this.#inventory_overlay;
        let available = 0;
        available += is_in_inventory ? this.#internal_inventory[item_id] : 0;
        available += has_overlay ? this.#inventory_overlay[item_id] : 0;

        return available;
    }
    adjustInventory(item_id: ItemID, adjustment_delta: number): void {
        if (!(item_id in this.#inventory_overlay)) {
            this.#inventory_overlay[item_id] = 0;
        }
        this.#inventory_overlay[item_id] += adjustment_delta;
    }

    resetInventoryAdjustments(): void {
        this.#inventory_overlay = {};
    }
}
export { RunConfiguration };
