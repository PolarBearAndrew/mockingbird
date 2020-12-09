import { extendTheme } from '@chakra-ui/react'

export const ThemeMockingbird = extendTheme({
  colors: {
    mockingBirdBlue: {
      100: '#006fd6',
      200: '#0065c2',
      300: '#005aad',
      400: '#005099',
      500: '#004585',
      600: '#003a70',
      700: '#00305c',
      800: '#002447',
      900: '#001a33'
    },
    mockingBirdPink: {
      100: '#f0b6e4',
      200: '#d698c9',
      300: '#cc87bc',
      400: '#b56da5',
      500: '#b55ba2',
      600: '#b5489d',
      700: '#b53699',
      800: '#b52496',
      900: '#b51292'
    }
  },
  styles: {
    global: {
      body: {
        bg: 'gray.200',
        color: 'gray.800',
      },
    }
  },
  components: {
    Heading: {
      baseStyle: {
        bg: '',
        color: 'white'
      }
    }
  }
})