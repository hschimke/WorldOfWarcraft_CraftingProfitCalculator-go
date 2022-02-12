export function SourceLink({text}: {text:string}){
    const source_uri = 'https://github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator';
    return(
        <a href={source_uri}>{text}</a>
    );
}
export function BugReportLink({text}: {text:string}){
    const bug_uri = 'https://github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator/issues';
    return(
        <a href={bug_uri}>{text}</a>
    );
}
export function CopyRightTag(){
    return(
        <span>Copyright 2021</span>
    );
}
export function ReadMeLink({text}: {text:string}){
    const readme_uri = 'https://github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator/blob/main/README.md';
    return(
        <a href={readme_uri}>{text}</a>
    );
}
export function AddonDownloadLink({text}: {text:string}){
    const addon_uri = '/addon-download';
    return(
        <a href={addon_uri}>{text}</a>
    );
}