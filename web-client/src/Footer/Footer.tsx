import styles from './Footer.module.css';
import {BugReportLink,CopyRightTag,SourceLink} from '../Shared/Links';

function Footer() {
    return (
        <footer className={styles.footer}>
            <ul>
                <li><CopyRightTag /></li>
                <li><SourceLink text="Source" /></li>
                <li><BugReportLink text="Report Bugs" /></li>
            </ul>
            <p>"World of Warcraft" is a trademark of Activision Blizzard. This is only a fansite, there is no affiliation between wowcpc.info and Activision Blizzard.</p>
        </footer>
    )
}

export default Footer;