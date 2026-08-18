if(_root._currentframe == 9)
{
   _root.menup.swapDepths(1);
   removeMovieClip(_root.menup);
   delete _root.menup.onEnterFrame;
}
if(_root._currentframe == 10)
{
   _root.stopallmusic();
   if(_root.savedata2.data.musicON)
   {
      _root.music222.start(0.1,100);
   }
   _root.returntomenu();
   delete _root.onEnterFrame;
}
if(_root._currentframe == 8)
{
   _root.gunlib.swapDepths(1);
   removeMovieClip(_root.gunlib);
   delete _root.gunlib.onEnterFrame;
}
if(_root._currentframe == 7)
{
   _root.menu_credits.swapDepths(1);
   removeMovieClip(_root.menu_credits);
   delete _root.menu_credits.onEnterFrame;
}
if(_root._currentframe == 6)
{
   _root.pgs.swapDepths(1);
   removeMovieClip(_root.pgs);
   delete _root.pgs.onEnterFrame;
}
if(_root._currentframe == 5)
{
   Key.removeListener(_root.menu_options.keyListener);
   _root.menu_options.swapDepths(1);
   removeMovieClip(_root.menu_options);
   delete _root.menu_options.onEnterFrame;
}
if(_root._currentframe == 4)
{
   _root.menu_campaign.swapDepths(1);
   removeMovieClip(_root.menu_campaign);
   delete _root.menu_campaign.onEnterFrame;
}
_root.gotoAndPlay(targetframe);
_root._x = 0;
_root._y = 0;
_X = 0;
_Y = 0;
_xscale = 100;
_yscale = 100;
