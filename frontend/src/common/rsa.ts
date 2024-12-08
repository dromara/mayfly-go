import openApi from './openApi';
import JSEncrypt from 'jsencrypt';
import { notBlank } from './assert';

var encryptor: any = null;

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
export async function RsaEncrypt(value: any) {
    // 不存在则返回空值
    if (!value) {
        return '';
    }
    if (encryptor != null && sessionStorage.getItem('RsaPublicKey') != null) {
        return encryptor.encrypt(value);
    }
    encryptor = new JSEncrypt();
    const publicKey = (await getRsaPublicKey()) as string;
    notBlank(publicKey, '获取公钥失败');
    encryptor.setPublicKey(publicKey); //设置公钥
    return encryptor.encrypt(value);
}
