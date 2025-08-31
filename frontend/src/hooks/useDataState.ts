import { ref } from 'vue';

export function useDataState<KeyType, ValueType extends number | boolean | string>() {
    const dataState = ref(new Map<KeyType, ValueType>());

    const setState = (key: KeyType, value: ValueType) => {
        dataState.value.set(key, value as any);
    };

    const getState = (key: KeyType): ValueType => {
        const result = dataState.value.get(key);
        return result as ValueType;
    };

    return {
        dataState,
        setState,
        getState,
    };
}
