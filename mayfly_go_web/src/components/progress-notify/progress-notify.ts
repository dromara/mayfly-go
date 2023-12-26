export const buildProgressProps = (): any => {
    return {
        progress: {
            title: {
                type: String,
            },
            executedStatements: {
                type: Number,
            },
        },
    };
};
