export interface Product {
    id: number
    name: string
    price: number
    stock: number
    category_id: number
    category_name?: string
}

export interface ProductInput {
    name: string
    price: number
    stock: number
    category_id: number
}

export interface Category {
    id: number
    name: string
    description: string
}

export interface CategoryInput {
    name: string
    description: string
}

export interface CartItem {
    product: Product
    qty: number
}

export interface CheckoutRequest {
    items: { product_id: number; quantity: number }[]
}

export interface Transaction {
    id: number
    total_amount: number
    created_at: string
    details: TransactionDetail[]
}

export interface TransactionDetail {
    id: number
    transaction_id: number
    product_id: number
    product_name?: string
    quantity: number
    subtotal: number
}

export interface SalesSummary {
    total_revenue: number
    total_transaksi: number
    produk_terlaris?: {
        nama: string
        qty_terjual: number
    }
}
