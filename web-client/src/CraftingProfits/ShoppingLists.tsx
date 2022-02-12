import './ShoppingLists.css';
import { AHItemPrice, VendorItemPrice } from '../Shared/GoldFormatter';

export interface ShoppingListsProps {
    name: string,
    lists: OutputFormatShoppingList
}

export interface ShoppingListProps {
    level: string | number,
    list: ShoppingList[]
}

export interface ShoppingListItemProps {
    item: ShoppingList
}

function ShoppingLists(props: ShoppingListsProps) {
    return (
        <div className="ShoppingLists">
            <span className="ShoppingListsHeader">
                Shopping List For: {props.name}
            </span>
            <ul>
                {Object.keys(props.lists).map(list => {
                    return <ShoppingList key={list} list={props.lists[list]} level={list} />
                })}
            </ul>
        </div>
    );
}

function ShoppingList(props: ShoppingListProps) {
    return (
        <li className="ShoppingList">
            <span className="ShoppingListTitle">
                List for rank {props.level}
            </span>
            <ul>
                {props.list.map(list_item => {
                    return <ShoppingListItem key={JSON.stringify(list_item)} item={list_item} />
                })}
            </ul>
        </li>
    );
}

function ShoppingListItem(props: ShoppingListItemProps) {
    const li = props.item;
    const show_vendor = (li.cost.vendor !== undefined);
    const show_ah = (li.cost.ah !== undefined);

    return (
        <li className="ShoppingListItem">
            <div className="ShoppingListColumn Quantity">
                {li.quantity.toLocaleString()}
            </div>
            <div className="ShoppingListColumn Data">
                {li.name} ({li.id})
                    {show_vendor &&
                    <VendorItemPrice vendor={li.cost.vendor} />
                }
                {show_ah &&
                    <AHItemPrice ah={li.cost.ah} />
                }
            </div>
        </li>
    );
}

export { ShoppingLists };