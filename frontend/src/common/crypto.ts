import { getToken } from '@/common/utils/storage';
import openApi from './openApi';
import { notBlank } from './assert';

/**
 * AES 加密数据
 * 与后端 cryptox.AesEncrypt 协议一致：CBC + Pkcs7，iv 为 key 前16字节，密文不携带 iv
 * @param word
 * @param key
 */
export async function AesEncrypt(word: string, key?: string): Promise<string> {
    if (!key) {
        key = getToken().substring(0, 16);
    }
    if (!key) {
        throw new Error('AES key is empty');
    }

    const encoder = new TextEncoder();
    const keyData = encoder.encode(key);
    const iv = keyData.slice(0, 16);

    const cryptoKey = await crypto.subtle.importKey('raw', keyData, { name: 'AES-CBC' }, false, ['encrypt']);
    const encrypted = await crypto.subtle.encrypt({ name: 'AES-CBC', iv }, cryptoKey, encoder.encode(word));

    return arrayBufferToBase64(encrypted);
}

/**
 * AES 解密数据
 * 与后端 cryptox.AesDecrypt 协议一致：iv 为 key 前16字节
 * @param word Base64 编码的密文
 * @param key
 */
export async function AesDecrypt(word: string, key?: string): Promise<string> {
    if (!key) {
        key = getToken().substring(0, 16);
    }
    if (!key) {
        throw new Error('AES key is empty');
    }

    const encoder = new TextEncoder();
    const keyData = encoder.encode(key);
    const iv = keyData.slice(0, 16);

    const data = base64ToArrayBuffer(word);

    const cryptoKey = await crypto.subtle.importKey('raw', keyData, { name: 'AES-CBC' }, false, ['decrypt']);
    const decrypted = await crypto.subtle.decrypt({ name: 'AES-CBC', iv }, cryptoKey, data as BufferSource);

    return new TextDecoder().decode(decrypted);
}

/** ArrayBuffer 转 Base64（分块处理避免调用栈溢出） */
function arrayBufferToBase64(buffer: ArrayBuffer): string {
    const bytes = new Uint8Array(buffer);
    const chunkSize = 0x8000; // 32KB per chunk
    let binary = '';
    for (let i = 0; i < bytes.length; i += chunkSize) {
        binary += String.fromCharCode.apply(null, bytes.subarray(i, i + chunkSize) as unknown as number[]);
    }
    return btoa(binary);
}

/** Base64 转 ArrayBuffer */
function base64ToArrayBuffer(base64: string): Uint8Array {
    const binary = atob(base64);
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) {
        bytes[i] = binary.charCodeAt(i);
    }
    return bytes;
}

/** PEM 格式公钥转 CryptoKey（SPKI 格式） */
async function pemToCryptoKey(pem: string): Promise<CryptoKey> {
    // 移除 PEM 头尾标记，提取 base64 内容
    const base64 = pem
        .replace(/-----BEGIN PUBLIC KEY-----/g, '')
        .replace(/-----END PUBLIC KEY-----/g, '')
        .replace(/\s/g, '');
    const der = base64ToArrayBuffer(base64);
    return crypto.subtle.importKey('spki', der as BufferSource, { name: 'RSA-PKCS1-v1_5', hash: 'SHA-256' }, false, ['encrypt']);
}

let cachedCryptoKey: CryptoKey | null = null;

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
 * 与后端 cryptox.RsaEncrypt 协议一致：RSA-PKCS1-v1_5
 *
 * @param value value
 * @returns 加密后的 Base64 值
 */
export async function RsaEncrypt(value: string): Promise<string> {
    if (!value) {
        return '';
    }

    // 获取或缓存 CryptoKey
    if (!cachedCryptoKey) {
        const publicKey = (await getRsaPublicKey()) as string;
        notBlank(publicKey, '获取公钥失败');
        cachedCryptoKey = await pemToCryptoKey(publicKey);
    }

    const encoder = new TextEncoder();
    const encrypted = await crypto.subtle.encrypt({ name: 'RSA-PKCS1-v1_5' }, cachedCryptoKey, encoder.encode(value));
    return arrayBufferToBase64(encrypted);
}
