import styles from  './GoldFormatter.module.css';

export interface GoldFormatterProps {
    raw_price: number
}

function GoldFormatter(props: GoldFormatterProps) {
    const price_in = props.raw_price;
    const price = Math.trunc(price_in);
    const copper = price % 100;
    const silver = (((price % 10000) - copper)) / 100;
    const gold = (price - (price % 10000)) / 10000;
    return (
        <span className={styles.PriceData}>
            <span className={styles.Gold}>
                {gold.toLocaleString()}
                <span className={styles.Currency}>g</span>
            </span>
            <span className={styles.Silver}>
                {silver.toLocaleString()}
                <span className={styles.Currency}>s</span>
            </span>
            <span className={styles.Copper}>
                {copper.toLocaleString()}
                <span className={styles.Currency}>c</span>
            </span>
        </span>
    );
}

function AHItemPrice(props: { ah: OutputFormatPrice }) {
    return (
        <div className={styles.AHItemPrice}>
            AH {props.ah.sales}: <GoldFormatter raw_price={props.ah.high} />/<GoldFormatter raw_price={props.ah.low} />/<GoldFormatter raw_price={props.ah.average} />
        </div>
    );
}

function VendorItemPrice(props: { vendor: number }) {
    return (
        <div className={styles.VendorItemPrice}>
            Vendor <GoldFormatter raw_price={props.vendor} />
        </div>
    );
}

export { GoldFormatter, AHItemPrice, VendorItemPrice };