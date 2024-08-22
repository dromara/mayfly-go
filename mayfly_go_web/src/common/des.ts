import CryptoJS from 'crypto-js';
import { getToken } from '@/common/utils/storage';

/**
 * AES 加密数据
 * @param word
 * @param key
 */
export function DesEncrypt(word: string, key?: string) {
    if (!key) {
        key = getToken().substring(0, 24);
    }
    const srcs = CryptoJS.enc.Utf8.parse(word);
    const iv = CryptoJS.lib.WordArray.random(8); // 生成随机IV
    const encrypted = CryptoJS.TripleDES.encrypt(srcs, CryptoJS.enc.Utf8.parse(key), {
        iv: iv,
        mode: CryptoJS.mode.CBC,
        padding: CryptoJS.pad.Pkcs7,
    });

    return iv.concat(encrypted.ciphertext).toString(CryptoJS.enc.Base64);
}

/**
 * AES 解密 ：字符串 key iv  返回base64
 *  */
export function DesDecrypt(encryptedData: string, key?: string) {
    if (!key) {
        key = getToken().substring(0, 32);
    }
    // 解码Base64
    const ciphertext = CryptoJS.enc.Base64.parse(encryptedData);

    // 分离IV和加密数据
    const iv = ciphertext.clone();
    iv.sigBytes = 8;
    iv.clamp();
    ciphertext.words.splice(0, 2); // 移除IV
    ciphertext.sigBytes -= 8;

    const decrypted = CryptoJS.TripleDES.decrypt({ ciphertext } as any, CryptoJS.enc.Utf8.parse(key), {
        iv: iv,
        mode: CryptoJS.mode.CBC,
        padding: CryptoJS.pad.Pkcs7,
    });

    return decrypted.toString(CryptoJS.enc.Utf8);
}
