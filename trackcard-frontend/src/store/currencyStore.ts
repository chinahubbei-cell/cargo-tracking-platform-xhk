import { create } from 'zustand';

export type CurrencyCode = 'CNY' | 'USD';

export interface CurrencyConfig {
    code: CurrencyCode;
    symbol: string;
    name: string;
}

export const CURRENCIES: Record<CurrencyCode, CurrencyConfig> = {
    CNY: { code: 'CNY', symbol: '¥', name: '人民币' },
    USD: { code: 'USD', symbol: '$', name: '美元' },
};

interface CurrencyState {
    currency: CurrencyCode;
    setCurrency: (currency: CurrencyCode) => void;
    getCurrencyConfig: () => CurrencyConfig;
    formatAmount: (amount: number | null | undefined) => string;
}

const STORAGE_KEY = 'globalCurrency';

export const useCurrencyStore = create<CurrencyState>((set, get) => ({
    currency: (localStorage.getItem(STORAGE_KEY) as CurrencyCode) || 'CNY',

    setCurrency: (currency: CurrencyCode) => {
        localStorage.setItem(STORAGE_KEY, currency);
        set({ currency });
    },

    getCurrencyConfig: () => {
        return CURRENCIES[get().currency];
    },

    formatAmount: (amount: number | null | undefined) => {
        if (amount == null) return '-';
        const config = CURRENCIES[get().currency];
        return `${config.symbol}${amount.toLocaleString('zh-CN', {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2
        })}`;
    },
}));
