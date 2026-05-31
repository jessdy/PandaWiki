'use client';
import ConsultDetail from '@/views/consult/ConsultDetail';
import { useParams } from 'next/navigation';

const Page = () => {
  const params = useParams<{ id: string }>();
  const id = parseInt(params?.id || '0', 10);
  if (!id || Number.isNaN(id)) {
    return null;
  }
  return <ConsultDetail id={id} />;
};

export default Page;
