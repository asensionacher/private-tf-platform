import { useMemo } from 'react';
import Convert from 'ansi-to-html';

interface AnsiOutputProps {
    content: string;
}

const convert = new Convert({
    fg: '#FFF',
    bg: '#000',
    newline: true,
    escapeXML: true,
    colors: {
        0: '#000',
        1: '#A00',
        2: '#0A0',
        3: '#A50',
        4: '#00A',
        5: '#A0A',
        6: '#0AA',
        7: '#AAA',
        8: '#555',
        9: '#F55',
        10: '#5F5',
        11: '#FF5',
        12: '#55F',
        13: '#F5F',
        14: '#5FF',
        15: '#FFF'
    }
});

export default function AnsiOutput({ content }: AnsiOutputProps) {
    const htmlContent = useMemo(() => {
        return convert.toHtml(content);
    }, [content]);

    return (
        <pre
            className="text-sm font-mono whitespace-pre-wrap break-words"
            style={{
                backgroundColor: '#1a1a1a',
                color: '#ffffff',
                padding: '1rem',
                borderRadius: '0.375rem',
                overflowX: 'auto'
            }}
            dangerouslySetInnerHTML={{ __html: htmlContent }}
        />
    );
}
