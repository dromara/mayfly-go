import forge from 'node-forge';
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

    const iv = key.substring(0, 16);
    const cipher = forge.cipher.createCipher('AES-CBC', forge.util.encodeUtf8(key));
    cipher.start({ iv: forge.util.encodeUtf8(iv) });
    cipher.update(forge.util.createBuffer(forge.util.encodeUtf8(word)));
    cipher.finish();
    return forge.util.encode64(cipher.output.getBytes());
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

    const iv = key.substring(0, 16);
    const decipher = forge.cipher.createDecipher('AES-CBC', forge.util.encodeUtf8(key));
    decipher.start({ iv: forge.util.encodeUtf8(iv) });
    decipher.update(forge.util.createBuffer(forge.util.decode64(word)));
    decipher.finish();
    return forge.util.decodeUtf8(decipher.output.getBytes());
}

let cachedRsaPublicKey: forge.pki.rsa.PublicKey | null = null;

/** 重置 RSA 加密缓存，使下次加密重新获取公钥 */
export function resetRsaCryptoKey() {
    cachedRsaPublicKey = null;
    sessionStorage.removeItem('RsaPublicKey');
}

export async function getRsaPublicKey() {
    let publicKey = sessionStorage.getItem('RsaPublicKey');
    if (publicKey) {
        return publicKey;
    }
    publicKey = (await openApi.getPublicKey()) as string;
    sessionStorage.setItem('RsaPublicKey', publicKey);
    return publicKey;
}

/** PEM 格式公钥解析为 forge PublicKey */
function parsePemPublicKey(pem: string): forge.pki.rsa.PublicKey {
    return forge.pki.publicKeyFromPem(pem) as forge.pki.rsa.PublicKey;
}

/**
 * 公钥加密指定值
 * 与后端 cryptox.RsaDecrypt 协议一致：RSA-OAEP + SHA-256
 *
 * @param value value
 * @returns 加密后的 Base64 值
 */
export async function RsaEncrypt(value: string): Promise<string> {
    if (!value) {
        return '';
    }

    if (!cachedRsaPublicKey) {
        const publicKeyPem = (await getRsaPublicKey()) as string;
        notBlank(publicKeyPem, '获取公钥失败');
        cachedRsaPublicKey = parsePemPublicKey(publicKeyPem);
    }

    const encrypted = cachedRsaPublicKey.encrypt(value, 'RSA-OAEP', {
        md: forge.md.sha256.create(),
        mgf1: { md: forge.md.sha256.create() },
    });
    return forge.util.encode64(encrypted);
}
