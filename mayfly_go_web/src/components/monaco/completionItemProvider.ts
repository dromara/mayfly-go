import * as monaco from 'monaco-editor/esm/vs/editor/editor.api';

/**
 * key: language, value: CompletionItemProvider
 */
const completionItemProviders: Map<string, any> = new Map();

export function registerCompletionItemProvider(language: string, completionItemProvider: any, replace: boolean = true) {
    const exist = completionItemProviders.get(language);
    if (exist) {
        if (!replace) {
            return;
        }
        exist.dispose();
    }
    completionItemProviders.set(language, monaco.languages.registerCompletionItemProvider(language, completionItemProvider));
}

export function dispposeCompletionItemProvider(language: string) {
    const exist = completionItemProviders.get(language);
    if (exist) {
        exist.dispose();
        completionItemProviders.delete(language);
    }
}
