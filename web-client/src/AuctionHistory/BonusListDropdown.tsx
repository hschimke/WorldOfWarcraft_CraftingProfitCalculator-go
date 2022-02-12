import { useContext } from 'react';
import { useSeenBonusesApi } from '../Shared/ApiClient';
import { AuctionHistoryDispatch } from './Shared';
import styles from './BonusListDropdown.module.css';

export interface BonusListDropdownProps {
    item: string,
    region: string,
    realm: string,
    ilevel: string,
    sockets: string,
    quality: string
}

function BonusListDropdown(props: BonusListDropdownProps) {
    const [apiState] = useSeenBonusesApi(props.item, props.region, props.realm);

    const raw = apiState.data;
    const mapped = raw !== undefined && raw.ERROR === undefined ? raw.mapped : [];
    const collected = raw !== undefined && raw.ERROR === undefined ? raw.collected : {
        ilvl: [],
        socket: [],
        quality: [],
        unknown: [],
        empty: true,
    };

    const dispatch = useContext(AuctionHistoryDispatch);

    const handleChange: React.ChangeEventHandler<HTMLSelectElement> = (event) => {
        if (dispatch !== undefined) {
            dispatch({ fieldName: event.target.name, value: event.target.value });
        }
    };

    return (
        <span>
            {false &&
                <label>
                    <select>
                        {mapped.map(bonus => {
                            return (
                                <option key={bonus.text}>{bonus.reduced}</option>
                            )
                        }
                        )}
                    </select>
                </label>
            }
            <label>
                Level
                <select onChange={handleChange} name="ilevel" value={props.ilevel}>
                    <option value="">Any</option>
                    {collected.ilvl.map(element => {
                        return (
                            <option key={element.id} value={element.id}>{element.level}</option>
                        );
                    })}
                </select>
            </label>
            <label>
                Socket
                    <select name="sockets" onChange={handleChange} value={props.sockets}>
                    <option value="">Any</option>
                    {collected.socket.map(element => {
                        return (
                            <option key={element.id} value={element.id}>{element.sockets}</option>
                        )
                    })}
                </select>
            </label>
            <label>
                Quality
                <select onChange={handleChange} name="quality" value={props.quality}>
                    <option value="">Any</option>
                    {collected.quality.map(element => {
                        return (
                            <option key={element.id} value={element.id}>{element.quality}</option>
                        );
                    })}
                </select>
            </label>
            {false  &&
                <pre>
                    {JSON.stringify(raw, undefined, 2)}
                </pre>
            }
        </span>
    );
}

export { BonusListDropdown };