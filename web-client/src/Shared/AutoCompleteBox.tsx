import { Suspense, useEffect, useState, useTransition, useRef, MutableRefObject } from "react";
import { fetchPromiseWrapper } from "./ApiClient";
import style from './AutoCompleteBox.module.css';

export type AutoCompleteAgg = { read: () => (string[] | undefined) };

function AutoCompleteBox({ source, filter, onSelect, currentValue, targetField }: { targetField: string, source: string, currentValue: string, filter?: string, onSelect: (field: string, value: string) => void }) {
    const [visible, setVisible] = useState(true);
    const [isLoading, onBatch] = useTransition();
    const [list, setList] = useState({ read: () => { return undefined; } } as AutoCompleteAgg);
    const [clickedValue, setClickedValue] = useState('');
    const ulRef = useRef<HTMLUListElement | null>(null);
    useOutsideAlerter(ulRef);

    useEffect(() => {
        const timer = setTimeout(() => {
            onBatch(() => {
                if (clickedValue !== currentValue) {
                    const endPoint = `${source}${filter !== undefined ? '?' + filter + '=' + encodeURIComponent(currentValue) : ''}`;
                    if (!visible) {
                        setVisible(true);
                    }
                    setList(fetchPromiseWrapper<string[]>(endPoint));
                }
            })
        }, 1000);

        return () => {
            clearTimeout(timer);
        }
    }, [currentValue]);

    const onClick = (event: string) => {
        setVisible(false);
        setClickedValue(event);
        onSelect(targetField, event);
    };

    return <>
        {visible &&
            <ul ref={ulRef} className={style.Box}>
                <Suspense fallback={<li>Loading</li>}>
                    <ItemList items={list} onSelect={onClick} />
                </Suspense>
            </ul>
        }
    </>

    function useOutsideAlerter(ref: MutableRefObject<HTMLUListElement | null>) {
        useEffect(() => {
            /**
             * Alert if clicked on outside of element
             */
            function handleClickOutside(event: MouseEvent) {
                if (ref.current && !ref.current.contains(event.target as Node)) {
                    setVisible(false);
                }
            }

            // Bind the event listener
            document.addEventListener("mousedown", handleClickOutside);
            return () => {
                // Unbind the event listener on clean up
                document.removeEventListener("mousedown", handleClickOutside);
            };
        }, [ref]);
    }
}

function ItemList({ items, onSelect }: { items: AutoCompleteAgg, onSelect: (event: string) => void }) {
    const item_list = items.read();
    if (item_list === undefined) {
        return null;
    }

    return <>
        {item_list.map((z) => {
            return <li className={style.Row} key={z} onClick={() => {
                onSelect(z);
            }}>{z}</li>
        })}
    </>
}

export { AutoCompleteBox };
