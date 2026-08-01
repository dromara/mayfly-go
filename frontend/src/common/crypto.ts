import CryptoJS from 'crypto-js';
import { getToken } from '@/common/utils/storage';
import openApi from './openApi';
import JSEncrypt from 'jsencrypt';
import { notBlank } from './assert';

/**
 * AES 加密数据
 * 与后端 cryptox.AesEncrypt 协议一致：CBC + Pkcs7，iv 为 key 前 16 字节，密文不携带 iv
 * @param word
 * @param key
 */
export function AesEncrypt(word: string, key?: string): string {
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

/**
 * AES 解密数据
 * 与后端 cryptox.AesDecrypt 协议一致：iv 为 key 前 16 字节
 * @param word
 * @param key
 */
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

let encryptor: JSEncrypt | null = null;

export async function getRsaPublicKey() {
    let publicKey = sessionStorage.getItem('RsaPublicKey');
    if (publicKey) {
        return publicKey;
    }
    publicKey = (await openApi.getPublicKey()) as string;
    sessionStorage.setItem('RsaPublicKey', publicKey);
    return publicKey;
}

/**
 * 公钥加密指定值
 *
 * @param value value
 * @returns 加密后的值
 */
export async function RsaEncrypt(value: string): Promise<string> {
    // 不存在则返回空值
    if (!value) {
        return '';
    }
    if (encryptor != null && sessionStorage.getItem('RsaPublicKey') != null) {
        const result = encryptor.encrypt(value);
        return result || '';
    }
    encryptor = new JSEncrypt();
    const publicKey = (await getRsaPublicKey()) as string;
    notBlank(publicKey, '获取公钥失败');
    encryptor.setPublicKey(publicKey); //设置公钥
    const result = encryptor.encrypt(value);
    return result || '';
}
