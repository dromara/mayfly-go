import CryptoJS from 'crypto-js';
import { getToken } from '@/common/utils/storage';

/**
 * AES 加密数据
 * @param word
 * @param key
 */
export function AesEncrypt(word: string, key?: string) {
    if (!key) {
        key = getToken().substring(0, 24);
    }

    const sKey = CryptoJS.enc.Utf8.parse(key);
    const encrypted = CryptoJS.AES.encrypt(word, sKey, {
        iv: sKey,
        mode: CryptoJS.mode.CBC,
        padding: CryptoJS.pad.Pkcs7,
    });

    return encrypted.ciphertext.toString(CryptoJS.enc.Base64);
}

export function AesDecrypt(word: string, key?: string): string {
    if (!key) {
        key = getToken().substring(0, 24);
    }

    const sKey = CryptoJS.enc.Utf8.parse(key);
    // key 和 iv 使用同一个值
    const decrypted = CryptoJS.AES.decrypt(word, sKey, {
        iv: sKey,
        mode: CryptoJS.mode.CBC, // CBC算法
        padding: CryptoJS.pad.Pkcs7, //使用pkcs7 进行padding 后端需要注意
    });

    return decrypted.toString(CryptoJS.enc.Base64);
}
