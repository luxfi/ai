// Lux brand tokens — the single source of truth for brand/network identity.
// Same shape as Hanzo/Zoo so the UI never hardcodes a brand string.
export interface Brand {
  name: string;
  productName: string;
  company: string;
  domain: string;
  appId: string;
  scheme: string;
  network: string;
  nativeToken: string;
  chainId: number;
  rpcUrl: string;
  explorerUrl: string;
  tagline: string;
}

export const BRAND: Brand = {
  name: 'Lux',
  productName: 'Lux Desktop',
  company: 'Lux Industries Inc',
  domain: 'lux.network',
  appId: 'network.lux.desktop',
  scheme: 'lux',
  network: 'LuxMainnet',
  nativeToken: 'LUX',
  chainId: 96369,
  rpcUrl: 'https://rpc.lux.network',
  explorerUrl: 'https://explorer.lux.network',
  tagline: 'Mine AI tokens by contributing GPU compute and chat with AI models on the Lux network.',
};

export default BRAND;
