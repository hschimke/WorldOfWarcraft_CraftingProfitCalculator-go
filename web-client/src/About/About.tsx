import { useEffect } from 'react';
import styles from './About.module.css';
import { BugReportLink, SourceLink, ReadMeLink, AddonDownloadLink } from '../Shared/Links';

function About() {
    useEffect(() => {
        document.title = "Crafting Profits Calculator - About";
    }, []);

    return (
        <div className={styles.About}>
            <span className={styles.header}>About</span>
            <p>Crafting Profits Calculator is designed to help figure out
                if it is worth your time and gold to craft an item or buy it on the auction house.</p>

            <span className={styles.header}>Known Issues</span>
            <ul>
                <li>The behavior of some international servers is undefined.</li>
                <li>China is not fully supported.</li>
                <li>Internationalization is not supported and all text is in en_us.</li>
                <li>All searches are done in en_us.</li>
                <li>Some items are incorrectly listed as being available from vendors,
                    this can confuse the system. If you find one of these please <BugReportLink text="report it" />.</li>
            </ul>

            <span className={styles.header}>Privacy</span>
            <p>wowcpc.info does not collect any personal information and requires no login.
                Basic server logs of errors may be kept, but ip addresses or other user identifiable information is not.
                You can check our work by looking at <SourceLink text="the source" />, which is exactly what we deploy.
            </p>
            <p>wowcpc.info uses Google Analytics to record page-views and metrics.
                We have disabled all data sharing and advertising features.
                In addition we have enabled anonymized ip-addresses.
            </p>


            <span className={styles.header}>Links</span>
            <ul>
                <li><ReadMeLink text="Full Readme [github]" /></li>
                <li><AddonDownloadLink text="Download Addon" /></li>
                <li><SourceLink text="Source Coce" /></li>
                <li><BugReportLink text="Report Issue" /></li>
            </ul>
        </div>
    )
}

export default About;