type LibraryUpdateHandler = () => void;
const libraryHandlers = new Set<LibraryUpdateHandler>();

export const dispatchLibraryUpdate = () => {
  libraryHandlers.forEach((handler) => handler());
};

export const subscribeLibraryUpdates = (handler: LibraryUpdateHandler) => {
  libraryHandlers.add(handler);
  return () => {
    libraryHandlers.delete(handler);
  };
};
