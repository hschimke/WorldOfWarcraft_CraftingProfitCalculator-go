function isCharacterProfession(candidate: unknown): candidate is CharacterProfession {
    if (typeof candidate !== 'string') {
        return false;
    }
    switch (candidate) {
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
            return true;
        default:
            return false;
    }
}

function validateProfessions(profession_list: Array<string>): Array<CharacterProfession> {
    return profession_list.filter(isCharacterProfession);
}

export { validateProfessions, isCharacterProfession };