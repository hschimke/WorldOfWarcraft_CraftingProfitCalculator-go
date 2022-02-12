import React, { Dispatch } from 'react';

const CraftingProfitsDispatch = React.createContext<Dispatch<any> | undefined>(undefined);
const all_professions: CharacterProfession[] = ['Jewelcrafting', 'Tailoring', 'Alchemy', 'Herbalism', 'Inscription', 'Enchanting', 'Blacksmithing', 'Mining', 'Engineering', 'Leatherworking', 'Skinning', 'Cooking'];

function validateAndCleanProfessions(profession_list: string[] | string): CharacterProfession[] | CharacterProfession | undefined {
    if (Array.isArray(profession_list)) {
        const return_array: Array<CharacterProfession> = [];

        for (const element of profession_list) {
            switch (element) {
                case 'Jewelcrafting':
                case 'Tailoring':
                case 'Alchemy':
                case 'Herbalism':
                case 'Inscription':
                case 'Enchanting':
                case 'Blacksmithing':
                case 'Mining':
                case 'Engineering':
                case 'Leatherworking':
                case 'Skinning':
                case 'Cooking':
                    return_array.push(element);
            }
        }

        return return_array;
    } else {
        const return_single = validateAndCleanProfessions([profession_list]);
        if (return_single !== undefined && return_single.length > 0) {
            if (Array.isArray(return_single)) {
                return return_single[0];
            } else {
                return return_single;
            }
        }
    }
}

export type CharacterProfessionList = CharacterProfession[];

export { CraftingProfitsDispatch, all_professions, validateAndCleanProfessions };