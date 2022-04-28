import { useState, useReducer, useEffect, Suspense, useTransition } from 'react';
import { RunForm } from './RunForm';
import { fetchPromiseWrapper } from '../Shared/ApiClient';
import RunResultDisplay, { RunResultDisplayQUEUE } from './RunResultDisplay';
import { all_professions, CraftingProfitsDispatch, CharacterProfessionList, validateAndCleanProfessions } from './Shared';
import { HelpBox } from './HelpBox';
import './RunCoordinator.css';

export interface RunCoordinatorFormDataReducerState {
    item: string,
    addon_data: string,
    required: string | number,
    region: string,
    realm: string,
    useAllProfessions: boolean,
    professions: CharacterProfessionList,
}

export interface RunCoordinatorFormDataReducerAction {
    field: string,
    value: string,
    professions: CharacterProfessionList
}

export interface RunCoordinatorProps { }

export type RunResultDataResponseAggregate = { read: () => RunReturn & ServerErrorReturn | undefined };
export type RunResultDataResponseAggregateQUEUE = { read: () => { job_id: string } | undefined };

const formDataReducer = (state: RunCoordinatorFormDataReducerState, action: RunCoordinatorFormDataReducerAction): RunCoordinatorFormDataReducerState => {
    switch (action.field) {
        case 'item':
            return { ...state, item: action.value };
        case 'addon_data':
            return { ...state, addon_data: action.value };
        case 'required':
            return { ...state, required: Number.isNaN(Number.parseInt(action.value)) ? 0 : Number.parseInt(action.value) };
        case 'region':
            return { ...state, region: action.value };
        case 'realm':
            return { ...state, realm: action.value };
        case 'useAllProfessions':
            return { ...state, useAllProfessions: action.value == "true" }
        case 'professions':
            const cleaned_prof = validateAndCleanProfessions(action.value);
            console.log(cleaned_prof);
            if (cleaned_prof !== undefined && !Array.isArray(cleaned_prof)) {
                const index = state.professions.indexOf(cleaned_prof);
                const new_profs = state.professions.slice();
                if (index > -1) {
                    new_profs.splice(index, 1);
                } else {
                    new_profs.push(cleaned_prof);
                }
                return { ...state, professions: new_profs };
            } else {
                return { ...state };
            }
        default:
            throw new Error();
    }
}

function RunCoordinator(props: RunCoordinatorProps) {
    useEffect(() => {
        document.title = "Crafting Profits Calculator";
    }, []);

    const [isFormTypeChangePending, startChangeFormType] = useTransition();
    const [isLoadingRun, startLoadingRun] = useTransition();

    //const [apiState, setPayload] = useFetchCPCApi();
    const [formData, dispatchFormUpdate] = useReducer(formDataReducer, {
        item: 'Grim-Veiled Bracers',
        addon_data: '',
        required: 1,
        region: 'US',
        realm: 'Hyjal',
        useAllProfessions: false,
        professions: all_professions.slice(),
    });
    const enable_run_button = true;//!apiState.isLoading;
    const output_display = '';
    //let output_display = apiState.isLoading ? `Analyzing ${formData.item}` : 'ready';

    //const raw_data = apiState.data?.ERROR === undefined ? apiState.data : undefined;
    /*
    if (apiState.data?.ERROR !== undefined) {
        output_display = apiState.data.ERROR;
    }
    */

    const [show_raw_results, updateShowRawResults] = useState(false);
    const [run_type, updateRunType] = useState('advanced');
    const [scanResult, setScanResult] = useState({ read: () => { return undefined; } } as RunResultDataResponseAggregateQUEUE);

    const handleSubmit: React.FormEventHandler = (event) => {
        startLoadingRun(() => {
            event.preventDefault();

            const run_data = {
                type: run_type === 'advanced' ? 'custom' : 'json',
                item_id: formData.item,
                addon_data: formData.addon_data,
                count: formData.required,
                region: formData.region,
                server: formData.realm,
                use_all_professions: formData.useAllProfessions,
                professions: formData.professions,//JSON.stringify(formData.professions),
            };
            //setPayload(run_data);
            //setScanResult(fetchPromiseWrapper<OutputFormatObject & ServerErrorReturn | undefined>('/json_output', run_data));
            setScanResult(fetchPromiseWrapper<{ job_id: string } | undefined>('/json_output_QUEUED', run_data));
        });
    };

    const pickForm: React.ChangeEventHandler<HTMLSelectElement | HTMLInputElement> = (e) => {
        startChangeFormType(() => {
            switch (e.target.name) {
                case 'formType':
                    updateRunType(e.target.value);
                    break;
                case 'includeRaw':
                    updateShowRawResults(!show_raw_results);
                    break;
                default:
                    break;
            }
        });
    };

    return (
        <div className="RunCoordinator">
            <form className="TypePicker">
                <label>
                    Run Type:
                    <select onChange={pickForm} value={run_type} name="formType">
                        <option value="advanced">Advanced</option>
                        <option value="simple">Simple</option>
                    </select>
                </label>
                <label>
                    Include Raw Output:
                    <input type="checkbox" name="includeRaw" checked={show_raw_results} onChange={pickForm} />
                </label>
            </form>
                <CraftingProfitsDispatch.Provider value={dispatchFormUpdate}>
                    <RunForm handleSubmit={handleSubmit}
                        form_type={run_type}
                        item={formData.item}
                        addon_data={formData.addon_data}
                        required={formData.required}
                        region={formData.region}
                        realm={formData.realm}
                        professions={formData.professions}
                        allProfessions={all_professions}
                        useAllProfessions={formData.useAllProfessions}
                        button_enabled={enable_run_button} />
                </CraftingProfitsDispatch.Provider>
            <HelpBox />
            <div className="ResultDisplays">
                {isLoadingRun && <p>Analyzing {formData.item}</p>}
                <Suspense fallback={<p>Analyzing {formData.item}</p>}>
                    <RunResultDisplayQUEUE item_name={formData.item} raw_run={scanResult} status={output_display} show_raw_result={show_raw_results} />
                </Suspense>
            </div>
        </div>
    );
}

export default RunCoordinator;