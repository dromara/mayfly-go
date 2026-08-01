import { languages } from 'monaco-editor';

/**
 * key: language, value: CompletionItemProvider
 */
const completionItemProviders: Map<string, { dispose: () => void }> = new Map();

export function registerCompletionItemProvider(language: string, completionItemProvider: any, replace: boolean = true) {
    const exist = completionItemProviders.get(language);
    if (exist) {
        if (!replace) {
            return;
        }
        exist.dispose();
    }
    completionItemProviders.set(language, languages.registerCompletionItemProvider(language, completionItemProvider));
}

export function dispposeCompletionItemProvider(language: string) {
    const exist = completionItemProviders.get(language);
    if (exist) {
        exist.dispose();
        completionItemProviders.delete(language);
    }
}
