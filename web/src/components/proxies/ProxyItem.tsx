import React from 'react';
import styled from 'styled-components'
import { Typography, Card, Tag } from 'antd'
import { IProxy } from '../../interfaces/Proxy';
import { EditOutlined, LoadingOutlined, DeleteOutlined } from '@ant-design/icons';
import { notifier } from '../../utils/notify';

interface Props {
  proxy: IProxy;
  onEditBtnClick: (proxyId: number) => void
  onDeleteBtnClick: (proxyId: number) => void
}

function ProxyItem(props: Props) {
  const { Text } = Typography;
  const { proxy } = props;
  const desc = proxy.desc || 'This proxy doesn\'t have any descriptions';

  const proxyStatus = proxy.status === 'active'
    ? <Text strong type='success'>Active</Text>
    : proxy.status === 'inactive'
      ? <Text>Inactive</Text>
      : proxy.status === 'updating'
        ? <LoadingOutlined style={{ fontSize: 13 }} spin />
        : <Text type='danger'>unknown</Text>

  return (
    <Card
      hoverable
      bordered={false}
      extra={(<div style={{ padding: '4px 12px' }}>{proxyStatus}</div>)}
      title={`${proxy.reqMethod} ${proxy.upstreamHost}:${proxy.upstreamPort}${proxy.prefix}`}
      style={{
        marginBottom: '24px',
        opacity: proxy.status === 'active' ? 1 : '.5'
      }}
      actions={[
        <span></span>,
        <span></span>,
        <span></span>,
        <span></span>,
        <span></span>,
        <span></span>,
        <DeleteOutlined key='delete' onClick={() => {
          if (!proxy.id) {
            notifier.warning('MockerId not found', '');
            return;
          }
          props.onDeleteBtnClick(proxy.id)
        }} />,
        <EditOutlined key="edit" onClick={() => {
          if (!proxy.id) {
            notifier.warning('MockerId not found', '');
            return;
          }
          props.onEditBtnClick(proxy.id);
        }} />
      ]}>
      <ProxyTagWrapper>
        <ProxyTag>
          <Tag>Proxy</Tag>
        </ProxyTag>
        <ProxyTag>
          <Tag>{`ID  ${proxy.id}`}</Tag>
        </ProxyTag>
      </ProxyTagWrapper>
      {desc.split('\n').map((str, i) => (
        <span key={i}>
          <Text type="secondary">{str}</Text>
          <br />
        </span>
      ))}
    </Card>
  );
}

const ProxyTagWrapper = styled.div`
  display: flex;
  width: 100%;
`

const ProxyTag = styled.span`
  margin: 0 4px 0 0;
`;

export default ProxyItem;
