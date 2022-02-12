import { useState } from 'react';
import styles from './HelpBox.module.css';

export function HelpBox() {
    const [showMe, updateShowMe] = useState(true);
    const showHide = () => {
        updateShowMe(!showMe);
    }

    if (!showMe) {
        return (<div className={styles.HelpBox} onClick={showHide}>
            <span className={styles.HelpTop}>Help</span>
        </div>);
    }

    return (
        <div className={styles.HelpBox} onClick={showHide}>
            <span className={styles.HelpTop}>Help</span>
            <span className={styles.HelpHeader}>Form</span>
            <p>Help filling out the form.
                Example values are provided throughout.
                Copy the outpout from the Addon into the box provided to perform a more automated run.</p>
            <span className={styles.HelpHeader}>Results</span>
            <p>
                Results are presented as a collapsable heirarchy of recipes and components.
            </p>
            <p>Click on the 'Components' or 'Recipes' links under the price output for more details.
                For example, for the default search there will be a box with the text 'Grim-Veiled Bracers190(42857)'.
                Two lines below you will see a line reading '3 COMPONENTS', click on that to expand the necessary parts of the recipe.
                One of the required items is listed as 'Enchanted Lightless Silk(172439)Requires 10', below that you will find a link
                for '1 RECIPES'.
                If you select that, you will see the recipe details for Enchanged Lightless Silk.
            </p>
            <span className={styles.HelpHeader}>Shopping List</span>
            <p>The shopping list shows what you will need to buy for each different recipe rank to produce the item.</p>
            <p>If inventory data is provided using the Addon then it will only show items you don't know, so only those which need to be purchased.</p>
        </div>
    )
}