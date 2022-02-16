import { useEffect, useState } from 'react';
import { textFriendlyOutputFormat } from '../Shared/text-output-helpers';
import './RunResultDisplay.css';
import { GoldFormatter, VendorItemPrice, AHItemPrice } from '../Shared/GoldFormatter';
import { ShoppingLists } from './ShoppingLists';
import type { RunResultDataResponseAggregate, RunResultDataResponseAggregateQUEUE } from './RunCoordinator';

const hidden_recipe_listing_header = {};

const hidden_run_result_header = {};

export interface RecipeListingProps {
    recipe: OutputFormatObject["recipes"][number]
}

export interface RunResultItemProps {
    raw_run: ServerRunResultReturn & ServerErrorReturn | ServerRunResultReturn | undefined,
    show_children?: boolean
}
export interface RunResultDisplayProps {
    raw_run: RunResultDataResponseAggregate,
    status: string,
    show_raw_result: boolean
}

export interface RunResultDisplayPropsQUEUE {
    raw_run: RunResultDataResponseAggregateQUEUE,
    item_name: string,
    status: string,
    show_raw_result: boolean
}

function RecipeListing(props: RecipeListingProps) {
    const [child_visible, setChildVisibility] = useState(false);

    const toggleChildren: React.MouseEventHandler = (e) => {
        setChildVisibility(!child_visible);
    };

    const show_ah_price = ((props.recipe.ah !== undefined) && (props.recipe.ah.sales > 0));
    const show_parts = (props.recipe.parts !== undefined);
    const parent_styles = child_visible ? {} : hidden_recipe_listing_header;
    const child_addins = child_visible ? '' : ' HiddenChild';
    return (
        <div className="RecipeListing" style={parent_styles}>
            <div className="RecipeHeader">
                <span className="RecipeBanner">
                    Recipe
                </span>
                <span className="RecipeHeaderDetails">
                    <span className="RecipeName">
                        {props.recipe.name}
                    </span>
                    <span className="RecipeRank">
                        {props.recipe.rank}
                    </span>
                    <span className="RecipeId">
                        ({props.recipe.id})
                    </span>
                </span>
                <span className="RecipeCost">
                    <GoldFormatter raw_price={props.recipe.high} />
                    /
                    <GoldFormatter raw_price={props.recipe.low} />
                    /
                    <GoldFormatter raw_price={props.recipe.average} />
                </span>
            </div>
            {show_ah_price &&
                <AHItemPrice ah={props.recipe.ah} />}
            <span className="RecipePartsBanner" onClick={toggleChildren}>
                {props.recipe.parts.length} Components
            </span>
            <div className={'HideableChild' + child_addins}>
                {show_parts &&
                    props.recipe.parts.map(part => {
                        return <RunResultItem key={part.id} raw_run={part} show_children={false} />;
                    })
                }
            </div>
        </div>
    );
}

function RunResultItem({ raw_run, show_children = true }: RunResultItemProps) {
    const [child_visibility, updateChildVisibility] = useState(show_children);

    if (raw_run === undefined || ((raw_run as ServerErrorReturn).ERROR !== undefined)) {
        return null;
    }

    const toggleChildren: React.MouseEventHandler = (e) => {
        updateChildVisibility(!child_visibility);
    };

    let ah_addin = false;
    let vendor_addin = false;
    let recipes = false;
    let bonuses = false;
    let shopping = false;
    const output_data = raw_run;

    ah_addin = ((output_data.ah !== undefined) && (output_data.ah.sales > 0));
    vendor_addin = (output_data.vendor > 0);
    recipes = (output_data.recipes !== undefined);
    bonuses = (output_data.bonus_prices !== undefined);
    shopping = ('shopping_lists' in output_data && Object.keys(output_data.shopping_lists).length > 0);

    const children_classes = child_visibility ? '' : ' HiddenChild';
    const parent_styles = child_visibility ? {} : hidden_run_result_header;

    return (
        <div className="RunResultItem">
            <div className="RunResultItemRecipes" style={parent_styles}>
                <div className="RunResultItemRecipesHeader">
                    <span className="ItemBanner">
                        Item
                    </span>
                    <span className="ItemName">
                        {output_data.name}
                    </span>
                    <span className="ItemId">
                        ({output_data.id})
                    </span>
                    <span className="Required">
                        Requires {output_data.required}
                    </span>
                </div>
                {ah_addin &&
                    <AHItemPrice ah={output_data.ah} />
                }
                {vendor_addin &&
                    <VendorItemPrice vendor={output_data.vendor} />
                }
                <span className="ItemRecipesBanner" onClick={toggleChildren}>
                    {output_data.recipes.length} Recipes
                </span>
                <div className={'RunResultItemRecipesChildren HideableChild' + children_classes}>
                    {recipes &&
                        output_data.recipes.map(recipe => {
                            return <RecipeListing key={recipe.id} recipe={recipe} />
                        })
                    }
                    {bonuses &&
                        output_data.bonus_prices.map(bonus_price => {
                            return (
                                <div className="Bonuses">
                                    {output_data.name} ({output_data.id}) iLvl {bonus_price.level}
                                    <AHItemPrice ah={bonus_price.ah} />
                                </div>
                            );
                        })
                    }
                </div>
            </div>
            {shopping &&
                <ShoppingLists lists={output_data.shopping_lists} name={output_data.name} />
            }
        </div>
    );
}

function RunResultDisplay(props: RunResultDisplayProps) {
    const SHOW_RES = props.show_raw_result;
    const raw_run = props.raw_run.read();
    let res;
    if (raw_run !== undefined) {
        res = textFriendlyOutputFormat(raw_run, 1);
    }
    return (
        <div className="RunResultDisplay">
            {props.status !== 'ready' &&
                <div className="Status">
                    {(raw_run?.ERROR !== undefined) && raw_run?.ERROR}
                </div>
            }
            {SHOW_RES &&
                <div className="RawResult">
                    <pre>
                        {res}
                    </pre>
                </div>
            }
            <div className="WebResult">
                <RunResultItem raw_run={raw_run} />
            </div>
        </div>
    );
}

function RunResultDisplayQUEUE(props: RunResultDisplayPropsQUEUE){
    const SHOW_RES = props.show_raw_result;
    let raw_run = props.raw_run.read();

    const job_id = (raw_run as any)?.job_id;

    const [st, setSt] = useState(undefined as (OutputFormatObject & ServerErrorReturn) | undefined);

    useEffect(() => {
        let timer = setTimeout(() => {
            queue_checker_functino();
        }, 10);
        const queue_checker_functino = () => {
            if (raw_run?.job_id !== undefined) {
                fetch('json_output_CHECK', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ job_id: job_id })
                }).then((intermediate) => {
                    intermediate.json().then((data) => {
                        if (data.job_id === undefined) {
                            setSt(data);
                        } else {
                            setSt(undefined);
                            timer = setTimeout(queue_checker_functino, 1000)
                        }
                    })
                });
            }
        };
        return () => {
            clearTimeout(timer);
        }
    }, [props.raw_run]);

    if(st === undefined && job_id !== undefined){
        return(<JobRunningBox job_id={job_id} item_name={props.item_name} />);
    }

    let res;
    if (st !== undefined) {
        res = textFriendlyOutputFormat(st, 1);
    }
    return (
        <div className="RunResultDisplay">
            {props.status !== 'ready' &&
                <div className="Status">
                    {(st?.ERROR !== undefined) && st?.ERROR}
                </div>
            }
            {SHOW_RES &&
                <div className="RawResult">
                    <pre>
                        {res}
                    </pre>
                </div>
            }
            <div className="WebResult">
                <RunResultItem raw_run={st} />
            </div>
        </div>
    );
}

function JobRunningBox({job_id, item_name}: {job_id: string, item_name?: string}){
    return(
        <div>
            {item_name &&
            <p>Job submitted for {item_name}</p>}
            <p>Job {job_id} is running.</p>
        </div>
    );
}

export default RunResultDisplay;
export { RunResultDisplayQUEUE };