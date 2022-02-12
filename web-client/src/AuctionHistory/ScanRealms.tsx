import { Suspense } from 'react';
import { BugReportLink } from '../Shared/Links';
import { fetchPromiseWrapper } from '../Shared/ApiClient';
import styles from './ScanRealms.module.css';

type ScanRealmsResult = { realm_names: string, realm_id: ConnectedRealmID, region: RegionCode }[];

const realms_dl = fetchPromiseWrapper<ScanRealmsResult>('/scanned_realms');

export function ScanRealms() {

    return <div className={styles.ScanRealmsList}>
        <p>You can enter any realm or region in the search box but you won't get any results unless it is one of the ones listed below.
            If you would like to have you realm added to the scan list, please <BugReportLink text="let us known" />.
        </p>
        <p>Scanning is enabled on the following realms:</p>
        <Suspense fallback={<p>LOADING...</p>}>
            <ScanRealmsList />
        </Suspense>
    </div>;
}

function ScanRealmsList() {
    const realms = realms_dl.read();
    return <ul>
        {realms.map((scan_target) => {
            return (<li key={scan_target.realm_id}>{scan_target.realm_names.slice(2)} in {scan_target.region}</li>);
        })}
    </ul>;
}