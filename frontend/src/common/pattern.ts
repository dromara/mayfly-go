export const AccountUsernamePattern = {
    pattern: /^[a-zA-Z0-9_]{5,20}$/g,
    message: '只允许输入5-20位大小写字母、数字、_-.:',
};

export const ResourceCodePattern = {
    pattern: /^[a-zA-Z0-9_\-.:]{1,32}$/g,
    message: '只允许输入1-32位大小写字母、数字、_-.:',
};
