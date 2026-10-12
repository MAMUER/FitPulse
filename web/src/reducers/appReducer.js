import { defaultState } from './appState';

export const initialState = { ...defaultState };

export function appReducer(state, action) {
  switch (action.type) {
    case 'UPDATE':
      return { ...state, ...action.payload };
    case 'SET_SCREEN':
      return { ...state, screen: action.payload };
    case 'SET_THEME':
      return { ...state, theme: action.payload };
    case 'SET_LANGUAGE':
      return { ...state, language: action.payload };
    case 'SET_PROFILE':
      return { ...state, profile: { ...state.profile, ...action.payload } };
    case 'SET_LOADING':
      return { ...state, loading: { ...state.loading, ...action.payload } };
    case 'CLEAR_TOAST':
      return { ...state, __toast: null, __toastTimer: null };
    default:
      return state;
  }
}
