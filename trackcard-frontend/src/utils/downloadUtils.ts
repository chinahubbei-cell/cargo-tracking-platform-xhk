import { saveAs } from 'file-saver';

/**
 * 通用文件下载工具函数
 * 使用 file-saver 处理文件下载，解决浏览器兼容性问题
 */

export const downloadFile = (blob: Blob, filename: string): void => {
    saveAs(blob, filename);
};

/**
 * 下载Excel文件
 * @deprecated 推荐直接使用 XLSX.writeFile
 */
export const downloadExcel = (buffer: ArrayBuffer, filename: string): void => {
    const blob = new Blob([buffer], {
        type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
    });
    const fullFilename = filename.endsWith('.xlsx') ? filename : `${filename}.xlsx`;
    saveAs(blob, fullFilename);
};

/**
 * 下载PNG图片
 */
export const downloadPng = (blob: Blob, filename: string): void => {
    const fullFilename = filename.endsWith('.png') ? filename : `${filename}.png`;
    saveAs(blob, fullFilename);
};
