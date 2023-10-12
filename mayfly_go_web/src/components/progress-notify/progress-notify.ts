export const buildProgressProps = (): any => {
    return {
        progress: {
            sqlFileName: {
                type: String,
            },
            executedStatements: {
                type: Number,
            },
        },
    };
};
