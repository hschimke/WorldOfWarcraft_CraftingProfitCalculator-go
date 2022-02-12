import {useState} from 'react';
import './RegionSelector.css';

const region_list = ['US', 'EU', 'KR', 'TW'];

export interface RegionSelectorProps {
    selected_region?: string,
    onChange: any,
    name?: string,
    label?: string
}

function RegionSelector({ selected_region = 'US', onChange, name = 'region', label = 'Region' } : RegionSelectorProps ) {
    const [selector_visible, updateVisible] = useState(false);

    const handlePick = (value : string) => {
        onChange({target:{name: name, value:value}});
        updateVisible(false);
    };

    const handleClick = (event : React.FormEvent) => {
        updateVisible(!selector_visible);
    }

    return (
        <label className="RegionSelector">
            {label}
            <select name={name} onChange={onChange} value={selected_region}>
                {region_list.map(r => {
                    return (
                        <option key={r} value={r}>{r}</option>
                    );
                })}
            </select>
            <div>
                <span onClick={handleClick}>{selected_region}</span>
                <ul style={selector_visible === false ? {display:'none'} : {}}>
                {region_list.map(r => {
                    return (
                        <li key={r} onClick={()=>{
                            handlePick(r);
                        }}>{r}</li>
                    );
                })}
                </ul>
            </div>
        </label>
    )
}

export { RegionSelector };